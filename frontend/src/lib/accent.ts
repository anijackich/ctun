export interface Lch {
  l: number
  c: number
  h: number
}

export interface Accent {
  stops: [Lch, Lch, Lch]
  fill: [Lch, Lch]
}

const BLUE: Lch = { l: 0.62, c: 0.2, h: 262 }
const CYAN: Lch = { l: 0.72, c: 0.15, h: 228 }
const VIOLET: Lch = { l: 0.62, c: 0.18, h: 290 }

const DEFAULT_ACCENT: Accent = { stops: [BLUE, CYAN, VIOLET], fill: [BLUE, CYAN] }

const MUTED_CHROMA = 0.02
const MAX_CHROMA = 0.22
const VIVID_CHROMA = 0.06
const RUN_BREAK = 0.09
const SAME_COLOR = 0.12
const DISTINCT_HUE = 30
const ANALOGOUS_SHIFT = 32
const SAMPLE_PX = 128
const SNAP_DISTANCE = 0.25
const MIN_CLUSTER_SHARE = 0.04
const WHITE_L = 0.82
const BLACK_L = 0.3
const INSET = 0.14

type Lab = [l: number, a: number, b: number]

export const muted = (accent: Accent): Accent => {
  const dull = (c: Lch): Lch => ({ ...c, c: MUTED_CHROMA })
  return {
    stops: accent.stops.map(dull) as Accent['stops'],
    fill: accent.fill.map(dull) as Accent['fill'],
  }
}

const srgbToLinear = (c: number) => {
  const v = c / 255
  return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
}

function toLab(r: number, g: number, b: number): Lab {
  const [lr, lg, lb] = [srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)]
  const l = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb)
  const m = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb)
  const s = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb)
  return [
    0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s,
  ]
}

const toLch = ([l, a, b]: Lab): Lch => ({
  l,
  c: Math.min(MAX_CHROMA, Math.hypot(a, b)),
  h: ((Math.atan2(b, a) * 180) / Math.PI + 360) % 360,
})

const distance = (x: Lab, y: Lab) => Math.hypot(x[0] - y[0], x[1] - y[1], x[2] - y[2])

const mean = (labs: Lab[]): Lab => {
  const sum = labs.reduce<Lab>((acc, v) => [acc[0] + v[0], acc[1] + v[1], acc[2] + v[2]], [0, 0, 0])
  return [sum[0] / labs.length, sum[1] / labs.length, sum[2] / labs.length]
}

const hueDistance = (a: number, b: number) => {
  const d = Math.abs(a - b) % 360
  return d > 180 ? 360 - d : d
}

function render(emoji: string): ImageData | null {
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = SAMPLE_PX
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return null
  ctx.font = `${SAMPLE_PX * 0.8}px "Apple Color Emoji", "Noto Color Emoji", "Segoe UI Emoji", sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(emoji, SAMPLE_PX / 2, SAMPLE_PX / 2)
  return ctx.getImageData(0, 0, SAMPLE_PX, SAMPLE_PX)
}

function flagGrid(image: ImageData): Lab[][] | null {
  const { data, width, height } = image
  let [x0, y0, x1, y1] = [width, height, -1, -1]
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      if (data[(y * width + x) * 4 + 3] > 200) {
        x0 = Math.min(x0, x)
        y0 = Math.min(y0, y)
        x1 = Math.max(x1, x)
        y1 = Math.max(y1, y)
      }
    }
  }
  if (x1 < 0) return null
  const dx = Math.round((x1 - x0) * INSET)
  const dy = Math.round((y1 - y0) * INSET)
  const grid: Lab[][] = []
  for (let y = y0 + dy; y <= y1 - dy; y++) {
    const row: Lab[] = []
    for (let x = x0 + dx; x <= x1 - dx; x++) {
      const i = (y * width + x) * 4
      if (data[i + 3] > 200) row.push(toLab(data[i], data[i + 1], data[i + 2]))
    }
    if (row.length) grid.push(row)
  }
  return grid.length ? grid : null
}

function stripeProfile(grid: Lab[][]): Lab[] {
  const rows = grid.map(mean)
  const width = Math.min(...grid.map((r) => r.length))
  const cols = Array.from({ length: width }, (_, x) => mean(grid.map((r) => r[x])))
  const variance = (p: Lab[]) => {
    const m = mean(p)
    return p.reduce((acc, v) => acc + distance(v, m) ** 2, 0) / p.length
  }
  return variance(rows) >= variance(cols) ? rows : cols
}

interface ColorGroup {
  lab: Lab
  weight: number
  position: number
}

function stripeColors(profile: Lab[]): ColorGroup[] {
  const runs: { labs: Lab[]; start: number }[] = []
  profile.forEach((lab, i) => {
    const current = runs.at(-1)
    if (current && distance(mean(current.labs), lab) < RUN_BREAK) current.labs.push(lab)
    else runs.push({ labs: [lab], start: i })
  })

  const groups: ColorGroup[] = []
  for (const run of runs) {
    const lab = mean(run.labs)
    const same = groups.find((g) => distance(g.lab, lab) < SAME_COLOR)
    if (same) same.weight += run.labs.length
    else groups.push({ lab, weight: run.labs.length, position: run.start })
  }
  return groups
}

/**
 * Actual colour clusters of the flag: vivid hue buckets plus white and black, largest first.
 * Used to snap stripe colours blended by shading/antialiasing back to real flag colours.
 */
function colorClusters(grid: Lab[][]): { vivid: Lab[]; all: Lab[] } {
  const pixels = grid.flat()
  const buckets = new Map<string, Lab[]>()
  const add = (key: string, lab: Lab) => buckets.set(key, [...(buckets.get(key) ?? []), lab])
  for (const lab of pixels) {
    const { c, h } = toLch(lab)
    if (c >= VIVID_CHROMA) add(`h${Math.floor(h / 30)}`, lab)
    else if (lab[0] >= WHITE_L) add('white', lab)
    else if (lab[0] <= BLACK_L) add('black', lab)
  }
  const ranked = [...buckets.entries()]
    .filter(([, labs]) => labs.length >= pixels.length * MIN_CLUSTER_SHARE)
    .sort(([, a], [, b]) => b.length - a.length)
  return {
    vivid: ranked.filter(([key]) => key.startsWith('h')).map(([, labs]) => mean(labs)),
    all: ranked.map(([, labs]) => mean(labs)),
  }
}

function snap(lab: Lab, clusters: Lab[]): Lab {
  let best: Lab | undefined
  for (const candidate of clusters) {
    if (!best || distance(candidate, lab) < distance(best, lab)) best = candidate
  }
  return best && distance(best, lab) < SNAP_DISTANCE ? best : lab
}

function pickFill(colors: Lch[]): Accent['fill'] {
  const vivid = colors.filter((c) => c.c >= VIVID_CHROMA).sort((a, b) => b.c - a.c)
  if (!vivid.length) return DEFAULT_ACCENT.fill
  const [first] = vivid
  const second = vivid.find((c) => hueDistance(c.h, first.h) >= DISTINCT_HUE)
  return [first, second ?? { ...first, h: (first.h + ANALOGOUS_SHIFT) % 360 }]
}

function extractAccent(emoji: string): Accent | null {
  const image = render(emoji)
  const grid = image && flagGrid(image)
  if (!grid) return null

  const clusters = colorClusters(grid)
  const isNew = (picked: { lab: Lab }[], lab: Lab) =>
    picked.every((p) => distance(p.lab, lab) >= SAME_COLOR)

  const picked: { lab: Lab; position: number }[] = []
  for (const group of stripeColors(stripeProfile(grid)).sort((a, b) => b.weight - a.weight)) {
    const lab = snap(group.lab, clusters.all)
    if (picked.length < 3 && isNew(picked, lab)) picked.push({ lab, position: group.position })
  }
  picked.sort((a, b) => a.position - b.position)

  for (const lab of clusters.vivid) {
    if (picked.length < 3 && isNew(picked, lab)) picked.push({ lab, position: Infinity })
  }
  if (!picked.length) return null
  const labs = picked.map((p) => p.lab)

  const colors = labs.map(toLch)
  const stops = [colors[0], colors[1] ?? colors[0], colors[2] ?? colors[0]] as Accent['stops']
  const fill = pickFill(colors)
  return fill === DEFAULT_ACCENT.fill && colors.every((c) => c.c < VIVID_CHROMA)
    ? null
    : { stops, fill }
}

const cache = new Map<string, Accent | null>()

export function accentForEmoji(emoji: string | undefined): Accent {
  if (!emoji) return DEFAULT_ACCENT
  if (!cache.has(emoji)) cache.set(emoji, extractAccent(emoji))
  return cache.get(emoji) ?? DEFAULT_ACCENT
}

export function nearestHue(from: number, target: number) {
  return from + (((target - from + 540) % 360) - 180)
}
