import { useFormat, useT } from '../i18n'
import { delayLevel } from '../lib/format'
import styles from './Latency.module.css'

const BARS = 4
const ACTIVE_BARS = { good: 4, fair: 3, poor: 1, timeout: 0, untested: 0 } as const

interface LatencyProps {
  delay?: number
  pending?: boolean
}

export function Latency({ delay, pending }: LatencyProps) {
  const t = useT()
  const format = useFormat()
  const level = pending ? 'untested' : delayLevel(delay)
  const label =
    level === 'untested' ? t.servers.untested : level === 'timeout' ? t.servers.timeout : format.delay(delay!)

  return (
    <span
      className={styles.latency}
      data-level={level}
      data-pending={pending || undefined}
      aria-busy={pending || undefined}
    >
      <span className={styles.bars} aria-hidden>
        {Array.from({ length: BARS }, (_, i) => (
          <span key={i} data-on={i < ACTIVE_BARS[level] || undefined} />
        ))}
      </span>
      <span className={`${styles.value} mono tabular`}>{pending ? null : label}</span>
    </span>
  )
}
