import { Power } from 'lucide-react'
import { useMemo, type CSSProperties } from 'react'
import type { CoreStatus } from '../lib/types'
import { useNow } from '../lib/useNow'
import styles from './TunnelButton.module.css'

const TICKS = 72
const IDLE_AMPLITUDE = 0.16

interface TunnelButtonProps {
  status: CoreStatus
  level: number
  label: string
  onToggle: () => void
}

const FLAG_STOPS = ['var(--flag-1)', 'var(--flag-2)', 'var(--flag-3)']
const BLEND_START = 0.7

const tickColor = (i: number) => {
  const position = (i / TICKS) * FLAG_STOPS.length
  const sector = Math.floor(position)
  const t = Math.max(0, (position - sector - BLEND_START) / (1 - BLEND_START))
  const eased = t * t * (3 - 2 * t)
  const from = FLAG_STOPS[sector]
  const to = FLAG_STOPS[(sector + 1) % FLAG_STOPS.length]
  return `color-mix(in oklab, ${from} ${Math.round((1 - eased) * 100)}%, ${to})`
}

const COLORS = Array.from({ length: TICKS }, (_, i) => tickColor(i))

function amplitudes(level: number, phase: number) {
  return Array.from({ length: TICKS }, (_, i) => {
    const shape =
      0.5 + 0.5 * Math.sin(i * 0.42 + phase * 0.9) * Math.cos(i * 0.17 - phase * 0.55)
    return 0.22 + level * (0.35 + 0.65 * shape)
  })
}

export function TunnelButton({ status, level, label, onToggle }: TunnelButtonProps) {
  const running = status === 'running'
  const busy = status === 'starting' || status === 'stopping'
  const now = useNow(1000, running)
  const amps = useMemo(
    () => (running ? amplitudes(level, now / 1000) : null),
    [running, level, now],
  )

  return (
    <div className={styles.dial} data-status={status}>
      {COLORS.map((color, i) => (
        <span
          key={i}
          className={styles.tick}
          style={
            {
              '--i': i,
              '--c': color,
              '--amp': amps ? amps[i] : IDLE_AMPLITUDE,
            } as CSSProperties
          }
        >
          <span className={styles.bar} />
        </span>
      ))}
      <button
        className={styles.button}
        disabled={busy}
        aria-pressed={running}
        aria-label={label}
        title={label}
        onClick={onToggle}
      >
        <Power size={30} strokeWidth={2.4} />
      </button>
    </div>
  )
}
