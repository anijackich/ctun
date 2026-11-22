import { ChevronDown, ChevronUp, CircleHelp } from 'lucide-react'
import { Popover } from 'radix-ui'
import { useEffect, useRef, useState } from 'react'
import { Button } from '../../components/Button'
import menu from '../../components/Menu.module.css'
import { Segmented } from '../../components/Segmented'
import { Switch } from '../../components/Switch'
import { useT } from '../../i18n'
import { routing as routingApi } from '../../lib/backend'
import type { EntryError, RouteTarget, RoutingProfile, RoutingProfileState } from '../../lib/types'
import styles from './RoutingProfile.module.css'

const SAVE_DELAY_MS = 600
const FINAL_TARGETS: RouteTarget[] = ['proxy', 'direct', 'block']
const MIN_ROWS = 3
const MAX_ROWS = 16

const comparable = (p: RoutingProfile) =>
  JSON.stringify({
    ...p,
    lists: p.lists.map((l) => {
      const entries = [...l.entries]
      while (entries.length && !entries.at(-1)?.trim()) entries.pop()
      return { ...l, entries }
    }),
  })

export function ProfileEditor({ state }: { state: RoutingProfileState }) {
  const t = useT()
  const [draft, setDraft] = useState(state.profile)
  const [errors, setErrors] = useState(state.errors)
  const latest = useRef(draft)
  const pending = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  // Adopt changes made elsewhere (tray, another edit) unless they only differ by what saving
  // normalizes, so the cursor is never yanked mid-typing.
  useEffect(() => {
    if (pending.current === undefined && comparable(state.profile) !== comparable(latest.current)) {
      latest.current = state.profile
      setDraft(state.profile)
    }
    setErrors(state.errors)
  }, [state])

  const save = (next: RoutingProfile) => {
    clearTimeout(pending.current)
    pending.current = undefined
    void routingApi.saveProfile(next).then((saved) => setErrors(saved.errors), () => {})
  }

  useEffect(
    () => () => {
      if (pending.current !== undefined) save(latest.current)
    },
    [],
  )

  const change = (next: RoutingProfile, debounce = false) => {
    latest.current = next
    setDraft(next)
    if (!debounce) {
      save(next)
      return
    }
    clearTimeout(pending.current)
    pending.current = setTimeout(() => save(next), SAVE_DELAY_MS)
  }

  const move = (index: number, by: -1 | 1) => {
    const lists = [...draft.lists]
    ;[lists[index], lists[index + by]] = [lists[index + by], lists[index]]
    change({ ...draft, lists })
  }

  return (
    <div className={styles.editor}>
      <label className={styles.setting}>
        <span>
          {t.routingProfile.lan}
          <span className={styles.settingHint}>{t.routingProfile.lanHint}</span>
        </span>
        <Switch
          checked={!draft.proxyLan}
          onCheckedChange={(direct) => change({ ...draft, proxyLan: !direct })}
          aria-label={t.routingProfile.lan}
        />
      </label>
      <p className={styles.hint}>{t.routingProfile.order}</p>
      {draft.lists.map((list, i) => (
        <ListCard
          key={list.target}
          target={list.target}
          entries={list.entries}
          errors={errors.filter((e) => e.target === list.target)}
          canMoveUp={i > 0}
          canMoveDown={i < draft.lists.length - 1}
          onMove={(by) => move(i, by)}
          onChange={(entries) =>
            change({ ...draft, lists: draft.lists.map((l, j) => (j === i ? { ...l, entries } : l)) }, true)
          }
        />
      ))}
      <p className={styles.hint}>{t.routingProfile.then}</p>
      <div className={styles.final}>
        <span>{t.routingProfile.final}</span>
        <Segmented
          aria-label={t.routingProfile.final}
          value={draft.final}
          options={FINAL_TARGETS.map((value) => ({ value, label: t.routingProfile.targets[value] }))}
          onChange={(final) => change({ ...draft, final })}
        />
      </div>
    </div>
  )
}

interface ListCardProps {
  target: RouteTarget
  entries: string[]
  errors: EntryError[]
  canMoveUp: boolean
  canMoveDown: boolean
  onMove: (by: -1 | 1) => void
  onChange: (entries: string[]) => void
}

function ListCard({ target, entries, errors, canMoveUp, canMoveDown, onMove, onChange }: ListCardProps) {
  const t = useT()
  const count = entries.filter((e) => e.trim() && !e.trim().startsWith('#')).length
  const rows = Math.min(MAX_ROWS, Math.max(MIN_ROWS, entries.length + 1))

  return (
    <section className={styles.card} data-target={target}>
      <header className={styles.cardHeader}>
        <span className={styles.dot} />
        <h3>{t.routingProfile.targets[target]}</h3>
        <span className={styles.targetHint}>{t.routingProfile.targetHints[target]}</span>
        <span className={`${styles.count} mono`}>{t.routingProfile.entries(count)}</span>
        <span className={styles.spacer} />
        <Button icon variant="plain" disabled={!canMoveUp} onClick={() => onMove(-1)} title={t.routingProfile.moveUp} aria-label={t.routingProfile.moveUp}>
          <ChevronUp size={15} />
        </Button>
        <Button icon variant="plain" disabled={!canMoveDown} onClick={() => onMove(1)} title={t.routingProfile.moveDown} aria-label={t.routingProfile.moveDown}>
          <ChevronDown size={15} />
        </Button>
      </header>
      <textarea
        className={`${styles.entries} mono`}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        rows={rows}
        placeholder={t.routingProfile.placeholder[target]}
        value={entries.join('\n')}
        onChange={(e) => onChange(e.target.value.split('\n'))}
        aria-label={t.routingProfile.targets[target]}
      />
      {errors.length > 0 && (
        <ul className={styles.errors}>
          {errors.map((e) => (
            <li key={`${e.line}:${e.entry}`}>
              <span className="mono">{t.routingProfile.line(e.line)}</span> {e.message}
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}

export function SyntaxHelp() {
  const t = useT()
  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <Button variant="plain">
          <CircleHelp size={14} />
          {t.routingProfile.syntaxTitle}
        </Button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={`${menu.content} ${styles.syntax}`} align="end" sideOffset={6}>
          {t.routingProfile.syntax.map(([sample, meaning]) => (
            <div key={sample} className={styles.syntaxRow}>
              <code className="mono selectable">{sample}</code>
              <span>{meaning}</span>
            </div>
          ))}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
