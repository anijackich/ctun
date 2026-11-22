import { useMemo } from 'react'
import { useApp } from '../store/app'
import { en } from './en'
import type { Locale } from './locale'
import { ru } from './ru'

export { LOCALES, type Locale } from './locale'

type Widen<T> = T extends (...args: infer A) => string
  ? (...args: A) => string
  : T extends readonly string[]
    ? readonly string[]
    : T extends string
      ? string
      : { [K in keyof T]: Widen<T[K]> }

export type Messages = Widen<typeof en>

const MESSAGES: Record<Locale, Messages> = { en, ru }

export const useT = () => MESSAGES[useApp((s) => s.locale)]

export function useFormat() {
  const locale = useApp((s) => s.locale)
  return useMemo(() => createFormat(locale), [locale])
}

function createFormat(locale: Locale) {
  const m = MESSAGES[locale]
  const relative = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  const clock = new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })

  const bytes = (value: number) => {
    const units = m.units.bytes
    if (value <= 0) return `0 ${units[0]}`
    const exp = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
    const scaled = value / 1024 ** exp
    return `${scaled.toFixed(exp === 0 || scaled >= 100 ? 0 : 1)} ${units[exp]}`
  }

  return {
    bytes,
    speed: (bytesPerSecond: number) => bytes(bytesPerSecond) + m.units.perSecond,
    delay: (ms: number) => `${ms} ${m.units.ms}`,
    clock: (timestamp: number) => clock.format(timestamp),
    relative(timestamp: number, now = Date.now()) {
      const minutes = Math.round((timestamp - now) / 60_000)
      if (Math.abs(minutes) < 1) return m.relative.justNow
      if (Math.abs(minutes) < 60) return relative.format(minutes, 'minute')
      const hours = Math.round(minutes / 60)
      if (Math.abs(hours) < 24) return relative.format(hours, 'hour')
      return relative.format(Math.round(hours / 24), 'day')
    },
  }
}
