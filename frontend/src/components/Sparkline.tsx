import { useId } from 'react'
import type { Traffic } from '../lib/types'
import styles from './Sparkline.module.css'

const WIDTH = 480
const HEIGHT = 120

interface SparklineProps {
  points: Traffic[]
  capacity: number
  height: number
}

function path(values: number[], capacity: number, max: number) {
  const step = WIDTH / (capacity - 1)
  const offset = (capacity - values.length) * step
  const coords = values.map((v, i) => [offset + i * step, HEIGHT - 2 - (v / max) * (HEIGHT - 6)])
  if (coords.length < 2) return { line: '', area: '' }
  let line = `M${coords[0][0]},${coords[0][1]}`
  for (let i = 1; i < coords.length; i++) {
    const [x0, y0] = coords[i - 1]
    const [x1, y1] = coords[i]
    const mx = (x0 + x1) / 2
    line += ` C${mx},${y0} ${mx},${y1} ${x1},${y1}`
  }
  return { line, area: `${line} L${WIDTH},${HEIGHT} L${coords[0][0]},${HEIGHT} Z` }
}

export function Sparkline({ points, capacity, height }: SparklineProps) {
  const id = useId()
  const max = Math.max(1, ...points.map((p) => Math.max(p.up, p.down))) * 1.15
  const down = path(
    points.map((p) => p.down),
    capacity,
    max,
  )
  const up = path(
    points.map((p) => p.up),
    capacity,
    max,
  )

  return (
    <svg
      className={styles.chart}
      style={{ height }}
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      preserveAspectRatio="none"
      aria-hidden
    >
      <defs>
        <linearGradient id={`${id}-fill`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--brand-b)" stopOpacity="0.35" />
          <stop offset="100%" stopColor="var(--brand-b)" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={`${id}-stroke`} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="var(--flag-1)" />
          <stop offset="50%" stopColor="var(--flag-2)" />
          <stop offset="100%" stopColor="var(--flag-3)" />
        </linearGradient>
      </defs>
      <path d={down.area} fill={`url(#${id}-fill)`} />
      <path d={down.line} fill="none" stroke={`url(#${id}-stroke)`} strokeWidth="1.6" />
      <path d={up.line} fill="none" className={styles.up} strokeWidth="1.2" />
    </svg>
  )
}
