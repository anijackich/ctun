import { useEffect, useRef } from 'react'
import { useApp } from '../store/app'
import { useActiveNode } from '../store/selectors'
import { accentForEmoji, muted, nearestHue, type Accent, type Lch } from './accent'
import { GENERIC_FLAG, parseServerName } from './flag'

const SLOTS = ['s1', 's2', 's3', 'f1', 'f2'] as const

const colorsOf = (accent: Accent): Lch[] => [...accent.stops, ...accent.fill]

export function useAccent() {
  const node = useActiveNode()
  const live = useApp((s) => s.status === 'running' || s.status === 'starting')
  const current = useRef<Lch[] | null>(null)

  useEffect(() => {
    const flag = node ? parseServerName(node.name).flag : GENERIC_FLAG
    const base = accentForEmoji(flag === GENERIC_FLAG ? undefined : flag)
    const target = colorsOf(live ? base : muted(base))
    const prev = current.current
    const next = prev ? target.map((c, i) => ({ ...c, h: nearestHue(prev[i].h, c.h) })) : target
    current.current = next

    const root = document.documentElement.style
    next.forEach((color, i) => {
      root.setProperty(`--${SLOTS[i]}-l`, color.l.toFixed(4))
      root.setProperty(`--${SLOTS[i]}-c`, color.c.toFixed(4))
      root.setProperty(`--${SLOTS[i]}-h`, color.h.toFixed(2))
    })
  }, [node, live])
}
