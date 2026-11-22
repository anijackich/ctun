import { RefreshCw } from 'lucide-react'
import { useCallback, useDeferredValue, useEffect, useMemo, useState } from 'react'
import { Button } from '../../components/Button'
import { PageEmpty } from '../../components/PageEmpty'
import { Switch } from '../../components/Switch'
import { useFormat, useT } from '../../i18n'
import { routing as routingApi } from '../../lib/backend'
import { DIRECT, isReject } from '../../lib/groups'
import type { Routing as RoutingData, Rule, RuleSet } from '../../lib/types'
import { useApp } from '../../store/app'
import styles from './Routing.module.css'

const POLL_MS = 2000

function useRouting() {
  const running = useApp((s) => s.status === 'running')
  const activeProfileId = useApp((s) => s.activeProfileId)
  const [data, setData] = useState<RoutingData>()

  const reload = useCallback(() => routingApi.get().then(setData, () => {}), [])

  useEffect(() => {
    void reload()
    if (!running) return
    const timer = setInterval(() => void reload(), POLL_MS)
    return () => clearInterval(timer)
  }, [running, activeProfileId, reload])

  const setRuleDisabled = (index: number, disabled: boolean) => {
    setData((d) => d && { ...d, rules: d.rules.map((r) => (r.index === index ? { ...r, disabled } : r)) })
    void routingApi.setRuleDisabled(index, disabled).finally(() => void reload())
  }

  return { data, reload, setRuleDisabled }
}

const matches = (rule: Rule, query: string) =>
  [rule.type, rule.payload, rule.target].some((field) => field.toLowerCase().includes(query))

export function ActiveRules({ query }: { query: string }) {
  const t = useT()
  const { data, reload, setRuleDisabled } = useRouting()
  const deferredQuery = useDeferredValue(query.trim().toLowerCase())

  const rules = useMemo(
    () => (data && deferredQuery ? data.rules.filter((r) => matches(r, deferredQuery)) : (data?.rules ?? [])),
    [data, deferredQuery],
  )

  if (!data) return <PageEmpty />
  if (data.rules.length === 0) return <PageEmpty>{t.routing.empty}</PageEmpty>
  return (
    <div className={styles.content}>
          <ModeNotice />
          {!data.live && <p className={styles.notice}>{t.routing.offline}</p>}
          {data.ruleSets.length > 0 && <RuleSets sets={data.ruleSets} live={data.live} onUpdated={reload} />}
          <section className={styles.section}>
            <h2 className={styles.heading}>
              {t.routing.rules}
              <span className="mono">{data.rules.length}</span>
              {data.live && <span className={styles.headingHint}>{t.routing.toggleHint}</span>}
            </h2>
            {rules.length === 0 ? (
              <p className={styles.notice}>{t.common.noMatch}</p>
            ) : (
              <RuleTable rules={rules} live={data.live} onToggle={setRuleDisabled} />
            )}
          </section>
    </div>
  )
}

function ModeNotice() {
  const t = useT()
  const mode = useApp((s) => s.settings?.routingMode)
  const setRoutingMode = useApp((s) => s.setRoutingMode)
  if (mode !== 'global' && mode !== 'direct') return null
  return (
    <div className={`${styles.notice} ${styles.mode}`}>
      <span>{t.routing.mode[mode]}</span>
      <Button variant="primary" onClick={() => setRoutingMode('rule')}>
        {t.routing.useRules}
      </Button>
    </div>
  )
}

function RuleSets({ sets, live, onUpdated }: { sets: RuleSet[]; live: boolean; onUpdated: () => void }) {
  const t = useT()
  const format = useFormat()
  const [updating, setUpdating] = useState<string[]>([])

  const update = async (names: string[]) => {
    setUpdating((current) => [...new Set([...current, ...names])])
    await Promise.allSettled(names.map((name) => routingApi.updateRuleSet(name)))
    setUpdating((current) => current.filter((n) => !names.includes(n)))
    onUpdated()
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.heading}>
        {t.routing.ruleSets}
        <span className="mono">{sets.length}</span>
        <span className={styles.spacer} />
        {live && (
          <Button variant="plain" disabled={updating.length > 0} onClick={() => void update(sets.map((s) => s.name))}>
            {t.routing.updateAll}
          </Button>
        )}
      </h2>
      <div className={styles.sets}>
        {sets.map((set) => {
          const busy = updating.includes(set.name)
          return (
            <div key={set.name} className={styles.set}>
              <div className={styles.setText}>
                <span className={`${styles.setName} mono`}>{set.name}</span>
                <span className={styles.setMeta}>
                  {[set.behavior, set.format, set.source].filter(Boolean).join(' · ')}
                  {set.rules > 0 && ` · ${t.routing.ruleCount(set.rules)}`}
                  {set.updatedAt > 0 && ` · ${t.routing.updated(format.relative(set.updatedAt))}`}
                </span>
              </div>
              {live && (
                <Button
                  icon
                  variant="plain"
                  disabled={busy}
                  onClick={() => void update([set.name])}
                  title={t.routing.update}
                  aria-label={t.routing.update}
                >
                  <RefreshCw size={14} className={busy ? styles.spinning : undefined} />
                </Button>
              )}
            </div>
          )
        })}
      </div>
    </section>
  )
}

interface RuleTableProps {
  rules: Rule[]
  live: boolean
  onToggle: (index: number, disabled: boolean) => void
}

function RuleTable({ rules, live, onToggle }: RuleTableProps) {
  const t = useT()
  const format = useFormat()
  return (
    <div className={styles.table} data-live={live || undefined}>
      <div className={`${styles.row} ${styles.head}`}>
        <span />
        <span>{t.routing.columns.type}</span>
        <span>{t.routing.columns.match}</span>
        <span>{t.routing.columns.target}</span>
        {live && <span className={styles.num}>{t.routing.columns.hits}</span>}
        {live && <span className={styles.num}>{t.routing.columns.lastHit}</span>}
        {live && <span />}
      </div>
      {rules.map((rule) => (
        <div key={rule.index} className={styles.row} data-disabled={rule.disabled || undefined}>
          <span className={`${styles.index} mono`}>{rule.index + 1}</span>
          <span className={`${styles.type} mono`}>{rule.type}</span>
          <span className={`${styles.payload} mono selectable`} title={rule.payload}>
            {rule.payload}
          </span>
          <Target name={rule.target} />
          {live && <span className={`${styles.num} mono`}>{rule.hits > 0 ? rule.hits.toLocaleString() : '—'}</span>}
          {live && (
            <span className={`${styles.num} ${styles.lastHit}`}>
              {rule.lastHit > 0 ? format.relative(rule.lastHit) : '—'}
            </span>
          )}
          {live && (
            <Switch
              checked={!rule.disabled}
              onCheckedChange={(on) => onToggle(rule.index, !on)}
              aria-label={t.routing.toggle}
              title={t.routing.toggle}
            />
          )}
        </div>
      ))}
    </div>
  )
}

function Target({ name }: { name: string }) {
  const t = useT()
  if (name === DIRECT) return <span className={styles.target} data-kind="direct">{t.common.direct}</span>
  if (isReject(name)) return <span className={styles.target} data-kind="reject">{t.routing.reject}</span>
  return (
    <span className={styles.target} title={name}>
      {name}
    </span>
  )
}
