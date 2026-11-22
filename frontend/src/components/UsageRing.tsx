import { useId } from 'react'
import styles from './UsageRing.module.css'

const R = 7
const C = 2 * Math.PI * R

export function UsageRing({ ratio }: { ratio?: number }) {
  const id = useId()
  const value = ratio === undefined ? 0 : Math.min(1, Math.max(0, ratio))
  return (
    <svg className={styles.ring} viewBox="0 0 18 18" aria-hidden>
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="var(--brand-a)" />
          <stop offset="100%" stopColor="var(--brand-b)" />
        </linearGradient>
      </defs>
      <circle cx="9" cy="9" r={R} className={styles.track} />
      {ratio !== undefined && (
        <circle
          cx="9"
          cy="9"
          r={R}
          stroke={value > 0.9 ? 'var(--red)' : `url(#${id})`}
          className={styles.value}
          strokeDasharray={C}
          strokeDashoffset={C * (1 - value)}
        />
      )}
    </svg>
  )
}
