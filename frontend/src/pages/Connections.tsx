import { ChevronDown, X } from 'lucide-react'
import { useDeferredValue, useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '../components/Button'
import { PageEmpty } from '../components/PageEmpty'
import { PageHeader } from '../components/PageHeader'
import { SearchField } from '../components/SearchField'
import { useFormat, useT } from '../i18n'
import { backend } from '../lib/backend'
import { DIRECT } from '../lib/groups'
import type { Connection } from '../lib/types'
import { useApp } from '../store/app'
import styles from './Connections.module.css'

const POLL_MS = 1000

type Sort = 'newest' | 'download' | 'upload'

interface Row extends Connection {
  upRate: number
  downRate: number
}

const compare: Record<Sort, (a: Row, b: Row) => number> = {
  newest: (a, b) => b.start - a.start,
  download: (a, b) => b.downRate - a.downRate || b.download - a.download,
  upload: (a, b) => b.upRate - a.upRate || b.upload - a.upload,
}

function useConnections(running: boolean) {
  const [rows, setRows] = useState<Row[]>([])
  const previous = useRef({ at: 0, totals: new Map<string, Connection>() })

  useEffect(() => {
    if (!running) {
      setRows([])
      previous.current = { at: 0, totals: new Map() }
      return
    }
    let alive = true
    const poll = async () => {
      const list = await backend.connections().catch(() => undefined)
      if (!alive || !list) return
      const now = Date.now()
      const { at, totals } = previous.current
      const seconds = at ? (now - at) / 1000 : 0
      const rate = (current: number, before?: number) =>
        seconds && before !== undefined ? Math.max(0, current - before) / seconds : 0
      setRows(
        list.map((c) => {
          const before = totals.get(c.id)
          return { ...c, upRate: rate(c.upload, before?.upload), downRate: rate(c.download, before?.download) }
        }),
      )
      previous.current = { at: now, totals: new Map(list.map((c) => [c.id, c])) }
    }
    void poll()
    const timer = setInterval(() => void poll(), POLL_MS)
    return () => {
      alive = false
      clearInterval(timer)
    }
  }, [running])

  const remove = (id?: string) => setRows((current) => (id ? current.filter((r) => r.id !== id) : []))
  return { rows, remove }
}

const ruleLabel = (c: Connection) => (c.rulePayload ? `${c.rule} · ${c.rulePayload}` : c.rule)

const matches = (c: Connection, query: string) =>
  [c.host, c.process, c.rule, c.rulePayload, ...c.chain].some((field) => field.toLowerCase().includes(query))

export function Connections() {
  const t = useT()
  const running = useApp((s) => s.status === 'running')
  const { rows, remove } = useConnections(running)
  const [sort, setSort] = useState<Sort>('newest')
  const [query, setQuery] = useState('')
  const deferredQuery = useDeferredValue(query.trim().toLowerCase())

  const visible = useMemo(() => {
    const filtered = deferredQuery ? rows.filter((r) => matches(r, deferredQuery)) : rows
    return [...filtered].sort(compare[sort])
  }, [rows, sort, deferredQuery])

  const close = (id: string) => {
    remove(id)
    void backend.closeConnection(id).catch(() => {})
  }
  const closeAll = () => {
    remove()
    void backend.closeAllConnections().catch(() => {})
  }

  const empty = !running ? t.connections.disconnected : rows.length === 0 ? t.connections.empty : t.common.noMatch

  return (
    <>
      <PageHeader title={t.nav.connections}>
        <SearchField className={styles.search} placeholder={t.connections.search} value={query} onChange={setQuery} />
        <Button onClick={closeAll} disabled={rows.length === 0}>
          {t.connections.closeAll}
        </Button>
      </PageHeader>
      {visible.length === 0 ? (
        <PageEmpty>{empty}</PageEmpty>
      ) : (
        <div className={styles.table}>
          <div className={`${styles.row} ${styles.head}`}>
            <span>
              {t.connections.columns.destination}
              <span className={`${styles.count} mono`}>{t.connections.count(rows.length)}</span>
            </span>
            <span>{t.connections.columns.route}</span>
            <SortHeader label={t.connections.columns.download} sort="download" current={sort} onSort={setSort} />
            <SortHeader label={t.connections.columns.upload} sort="upload" current={sort} onSort={setSort} />
            <SortHeader label={t.connections.columns.age} sort="newest" current={sort} onSort={setSort} />
            <span />
          </div>
          <div className={styles.rows}>
            {visible.map((row) => (
              <ConnectionRow key={row.id} row={row} onClose={() => close(row.id)} />
            ))}
          </div>
        </div>
      )}
    </>
  )
}

function ConnectionRow({ row, onClose }: { row: Row; onClose: () => void }) {
  const t = useT()
  const outbound = row.chain.at(-1) ?? DIRECT
  const via = row.chain.slice(0, -1)
  const age = Math.max(0, Math.floor((Date.now() - row.start) / 1000))

  return (
    <div className={styles.row}>
      <div className={styles.cell}>
        <span className={styles.primary} title={row.host}>
          {row.host}
          <span className={`${styles.port} mono`}>:{row.port}</span>
        </span>
        <span className={styles.secondary} title={row.processPath || undefined}>
          {row.network === 'udp' && <span className={`${styles.badge} mono`}>UDP</span>}
          {row.process || row.inbound}
          {row.pid > 0 && <span className={`${styles.pid} mono`}>{row.pid}</span>}
        </span>
      </div>
      <div className={styles.cell}>
        <span className={styles.primary} data-direct={outbound === DIRECT || undefined}>
          {outbound === DIRECT ? t.common.direct : outbound}
        </span>
        <span className={styles.secondary} title={[...via, ruleLabel(row)].join(' · ')}>
          {[...via, ruleLabel(row)].join(' · ')}
        </span>
      </div>
      <RateCell rate={row.downRate} total={row.download} />
      <RateCell rate={row.upRate} total={row.upload} />
      <span className={`${styles.num} ${styles.age} mono`}>{t.connections.age(age)}</span>
      <button className={styles.close} onClick={onClose} title={t.connections.close} aria-label={t.connections.close}>
        <X size={13} strokeWidth={2.4} />
      </button>
    </div>
  )
}

interface SortHeaderProps {
  label: string
  sort: Sort
  current: Sort
  onSort: (sort: Sort) => void
}

function SortHeader({ label, sort, current, onSort }: SortHeaderProps) {
  const t = useT()
  const active = sort === current
  return (
    <button
      className={styles.sortHeader}
      data-active={active || undefined}
      aria-pressed={active}
      title={t.connections.sortBy(label)}
      onClick={() => onSort(sort)}
    >
      {label}
      <ChevronDown size={11} strokeWidth={2.6} className={styles.sortIcon} />
    </button>
  )
}

function RateCell({ rate, total }: { rate: number; total: number }) {
  const format = useFormat()
  return (
    <div className={`${styles.cell} ${styles.num} mono`}>
      <span className={styles.rate} data-active={rate > 0 || undefined}>
        {rate > 0 ? format.speed(rate) : '—'}
      </span>
      <span className={styles.secondary}>{format.bytes(total)}</span>
    </div>
  )
}
