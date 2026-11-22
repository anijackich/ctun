export function formatUptime(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(Math.floor(total / 3600))}:${pad(Math.floor((total % 3600) / 60))}:${pad(total % 60)}`
}

export const daysUntil = (timestamp: number, now = Date.now()) =>
  Math.ceil((timestamp - now) / 86_400_000)

type DelayLevel = 'good' | 'fair' | 'poor' | 'timeout' | 'untested'

export function delayLevel(delay?: number): DelayLevel {
  if (delay === undefined) return 'untested'
  if (delay === 0) return 'timeout'
  if (delay < 150) return 'good'
  if (delay < 400) return 'fair'
  return 'poor'
}
