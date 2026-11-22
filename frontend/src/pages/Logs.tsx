import { ArrowDown, Trash2, WrapText } from 'lucide-react'
import { useDeferredValue, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Button } from '../components/Button'
import { PageEmpty } from '../components/PageEmpty'
import { PageHeader } from '../components/PageHeader'
import { SearchField } from '../components/SearchField'
import { FilterMenu } from '../components/FilterMenu'
import { useFormat, useT } from '../i18n'
import { backend } from '../lib/backend'
import type { LogLevel, LogLine } from '../lib/types'
import styles from './Logs.module.css'

const MAX_LINES = 2000
const WRAP_KEY = 'ctun.logs.wrap'
const FOLLOW_THRESHOLD_PX = 24

const LEVELS = ['all', 'info', 'warning', 'error'] as const
type LevelFilter = (typeof LEVELS)[number]
const SOURCES = ['all', 'core', 'app'] as const
type SourceFilter = (typeof SOURCES)[number]

const RANK: Record<LogLevel, number> = { debug: 0, info: 1, warning: 2, error: 3 }
const MIN_RANK: Record<LevelFilter, number> = { all: 0, info: 1, warning: 2, error: 3 }

interface Line extends LogLine {
  id: number
}

let nextId = 0
const withIds = (lines: LogLine[]): Line[] => lines.map((line) => ({ ...line, id: nextId++ }))
const keepLast = (lines: Line[]) => (lines.length > MAX_LINES ? lines.slice(-MAX_LINES) : lines)

function useLogLines() {
  const [lines, setLines] = useState<Line[]>([])

  useEffect(() => {
    let alive = true
    const off = backend.onLogs((batch) => setLines((prev) => keepLast(prev.concat(withIds(batch)))))
    void backend.logs().then((history) => {
      if (!alive) return
      const last = history.at(-1)?.time ?? 0
      setLines((live) => keepLast(withIds(history).concat(live.filter((l) => l.time > last))))
    })
    return () => {
      alive = false
      off()
    }
  }, [])

  const clear = () => {
    setLines([])
    void backend.clearLogs()
  }

  return { lines, clear }
}

function useWrapPreference() {
  const [wrap, setWrap] = useState(() => {
    try {
      return localStorage.getItem(WRAP_KEY) !== 'false'
    } catch {
      return true
    }
  })
  const toggle = () =>
    setWrap((current) => {
      try {
        localStorage.setItem(WRAP_KEY, String(!current))
      } catch {
      }
      return !current
    })
  return [wrap, toggle] as const
}

export function Logs() {
  const t = useT()
  const format = useFormat()
  const { lines, clear } = useLogLines()
  const [level, setLevel] = useState<LevelFilter>('all')
  const [source, setSource] = useState<SourceFilter>('all')
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query.trim().toLowerCase())
  const listRef = useRef<HTMLDivElement>(null)
  const [following, setFollowing] = useState(true)
  const [wrap, toggleWrap] = useWrapPreference()

  const visible = useMemo(
    () =>
      lines.filter(
        (line) =>
          RANK[line.level] >= MIN_RANK[level] &&
          (source === 'all' || line.source === source) &&
          (!deferredQuery || line.message.toLowerCase().includes(deferredQuery)),
      ),
    [lines, level, source, deferredQuery],
  )

  const filterLabel =
    [source !== 'all' && t.logs.sources[source], t.logs.levelShort[level]].filter(Boolean).join(' · ') ||
    t.logs.filter

  const scrollToEnd = () => {
    const list = listRef.current
    if (list) list.scrollTop = list.scrollHeight
  }

  useLayoutEffect(() => {
    if (following) scrollToEnd()
  }, [visible, following])

  const onScroll = () => {
    const list = listRef.current
    if (!list) return
    setFollowing(list.scrollHeight - list.scrollTop - list.clientHeight < FOLLOW_THRESHOLD_PX)
  }

  return (
    <>
      <PageHeader title={t.nav.logs}>
        <FilterMenu
          label={filterLabel}
          active={level !== 'all' || source !== 'all'}
          sections={[
            {
              label: t.logs.source,
              value: source,
              options: SOURCES.map((value) => ({ value, label: t.logs.sources[value] })),
              onChange: (value) => setSource(value as SourceFilter),
            },
            {
              label: t.logs.level,
              value: level,
              options: LEVELS.map((value) => ({ value, label: t.logs.levels[value] })),
              onChange: (value) => setLevel(value as LevelFilter),
            },
          ]}
        />
        <SearchField className={styles.search} placeholder={t.logs.search} value={query} onChange={setQuery} />
        <Button
          icon
          variant="plain"
          className={styles.toggle}
          aria-pressed={wrap}
          onClick={toggleWrap}
          title={t.logs.wrap}
          aria-label={t.logs.wrap}
        >
          <WrapText size={15} />
        </Button>
        <Button icon variant="plain" onClick={clear} disabled={lines.length === 0} title={t.logs.clear} aria-label={t.logs.clear}>
          <Trash2 size={15} />
        </Button>
      </PageHeader>
      <div className={styles.body}>
        {visible.length === 0 ? (
          <PageEmpty>{lines.length === 0 ? t.logs.empty : t.common.noMatch}</PageEmpty>
        ) : (
          <div ref={listRef} className={styles.list} data-wrap={wrap || undefined} onScroll={onScroll}>
            {visible.map((line) => (
              <div key={line.id} className={styles.line} data-level={line.level} data-source={line.source}>
                <span className={styles.time}>{format.clock(line.time)}</span>
                <span className={styles.origin}>{t.logs.origin[line.source]}</span>
                <span className={styles.tag}>{t.logs.tag[line.level]}</span>
                <span className={`${styles.message} selectable`}>{line.message}</span>
              </div>
            ))}
          </div>
        )}
        {!following && visible.length > 0 && (
          <button className={styles.latest} onClick={() => setFollowing(true)}>
            <ArrowDown size={13} strokeWidth={2.4} />
            {t.logs.latest}
          </button>
        )}
      </div>
    </>
  )
}
