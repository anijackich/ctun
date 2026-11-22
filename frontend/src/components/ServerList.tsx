import { ArrowUpDown, Check, Sparkles, Zap } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { useDeferredValue, useMemo, useState, type CSSProperties } from 'react'
import { useT } from '../i18n'
import { parseServerName } from '../lib/flag'
import { isGroupNode, primaryGroup, resolveNode } from '../lib/groups'
import type { ProxyGroup, ProxyNode } from '../lib/types'
import { useApp } from '../store/app'
import { Button } from './Button'
import { Latency } from './Latency'
import menu from './Menu.module.css'
import { Protocol } from './Protocol'
import { SearchField } from './SearchField'
import { Segmented } from './Segmented'
import styles from './ServerList.module.css'

const SORT_MODES = ['default', 'name', 'delay'] as const
type SortMode = (typeof SORT_MODES)[number]
const STAGGER_MS = 16
const STAGGER_LIMIT = 24
const NO_GROUPS: ProxyGroup[] = []
const NO_NAMES: string[] = []

export function ServerList({ profileId }: { profileId: string }) {
  const t = useT()
  const groups = useApp((s) => s.groups[profileId] ?? NO_GROUPS)
  const inUse = useApp((s) => s.activeProfileId === profileId)
  const measuring = useApp((s) => s.measuring[profileId] ?? NO_NAMES)
  const testLatency = useApp((s) => s.testLatency)
  const selectable = groups.filter((g) => g.type === 'select')
  const [groupName, setGroupName] = useState<string>()
  const group = groups.find((g) => g.name === groupName) ?? primaryGroup(groups)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SortMode>('default')
  const deferredQuery = useDeferredValue(query.trim().toLowerCase())

  if (!group) return null
  const serverCount = group.nodes.filter((n) => !isGroupNode(n)).length
  const isTesting = measuring.length > 0

  return (
    <section className={styles.section}>
      <header className={styles.heading}>
        <h2>{t.servers.title}</h2>
        <span className="mono">{serverCount}</span>
        <div className={styles.spacer} />
        <SortMenu value={sort} onChange={setSort} />
        <Button
          variant="plain"
          icon
          data-fx="zap"
          data-fx-on="press"
          data-busy={isTesting || undefined}
          className={styles.testButton}
          disabled={isTesting}
          title={t.servers.testLatency}
          aria-label={t.servers.testLatency}
          onClick={() => void testLatency(profileId, group.name)}
        >
          <Zap size={14} />
        </Button>
      </header>
      <SearchField placeholder={t.servers.search} value={query} onChange={setQuery} />
      {selectable.length > 1 && (
        <Segmented
          aria-label={t.servers.group}
          value={group.name}
          options={selectable.map((g) => ({ value: g.name, label: g.name }))}
          onChange={setGroupName}
        />
      )}
      <GroupNodes
        profileId={profileId}
        group={group}
        groups={groups}
        query={deferredQuery}
        sort={sort}
        showSelection={inUse}
        measuring={measuring}
      />
    </section>
  )
}

function SortMenu({ value, onChange }: { value: SortMode; onChange: (v: SortMode) => void }) {
  const t = useT()
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <Button
          variant="plain"
          icon
          data-fx="sort"
          data-fx-on="press"
          aria-label={t.servers.sort[value]}
          title={t.servers.sort[value]}
        >
          <ArrowUpDown size={14} />
        </Button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={menu.content} align="end" sideOffset={4}>
          <DropdownMenu.RadioGroup value={value} onValueChange={(v) => onChange(v as SortMode)}>
            {SORT_MODES.map((mode) => (
              <DropdownMenu.RadioItem key={mode} value={mode} className={menu.item}>
                <span className={menu.check}>
                  <DropdownMenu.ItemIndicator>
                    <Check size={13} strokeWidth={3} />
                  </DropdownMenu.ItemIndicator>
                </span>
                {t.servers.sort[mode]}
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

interface GroupNodesProps {
  profileId: string
  group: ProxyGroup
  groups: ProxyGroup[]
  query: string
  sort: SortMode
  showSelection: boolean
  measuring: string[]
}

function GroupNodes({ profileId, group, groups, query, sort, showSelection, measuring }: GroupNodesProps) {
  const t = useT()
  const selectProxy = useApp((s) => s.selectProxy)

  const [groupNodes, nodes] = useMemo(() => {
    const filtered = group.nodes.filter((n) => !query || n.name.toLowerCase().includes(query))
    const sorted =
      sort === 'name'
        ? [...filtered].sort((a, b) =>
            parseServerName(a.name).title.localeCompare(parseServerName(b.name).title),
          )
        : sort === 'delay'
          ? [...filtered].sort((a, b) => (a.delay || Infinity) - (b.delay || Infinity))
          : filtered
    return [sorted.filter(isGroupNode), sorted.filter((n) => !isGroupNode(n))]
  }, [group.nodes, query, sort])

  const selectable = group.type === 'select'
  const select = (name: string) => selectProxy(profileId, group.name, name)
  const pending = new Set(measuring)

  if (!groupNodes.length && !nodes.length) {
    return <p className={styles.empty}>{t.common.noMatch}</p>
  }

  return (
    <div className={styles.list} aria-label={group.name}>
      {groupNodes.map((n) => (
        <GroupRow
          key={n.name}
          node={n}
          resolved={resolveNode(groups, n.name)}
          selected={showSelection && group.now === n.name}
          pending={pending.size > 0}
          disabled={!selectable}
          onSelect={() => select(n.name)}
        />
      ))}
      {nodes.map((n, i) => (
        <ServerRow
          key={n.name}
          node={n}
          index={i}
          selected={showSelection && group.now === n.name}
          pending={pending.has(n.name)}
          disabled={!selectable}
          onSelect={() => select(n.name)}
        />
      ))}
    </div>
  )
}

interface RowProps {
  node: ProxyNode
  selected: boolean
  pending: boolean
  disabled: boolean
  onSelect: () => void
}

function GroupRow({ node, resolved, selected, pending, disabled, onSelect }: RowProps & { resolved?: ProxyNode }) {
  const t = useT()
  const label = node.name === 'AUTO' ? t.servers.auto : node.name
  return (
    <button
      className={`${styles.row} ${styles.auto}`}
      aria-pressed={selected}
      disabled={disabled}
      onClick={onSelect}
    >
      <span className={styles.flag}>
        <Sparkles size={14} />
      </span>
      <span className={styles.name}>
        {label}
        {resolved && <span className={styles.sub}>{parseServerName(resolved.name).title}</span>}
      </span>
      <Latency delay={resolved?.delay ?? node.delay} pending={pending} />
      <SelectedMark selected={selected} />
    </button>
  )
}

function ServerRow({ node, index, selected, pending, disabled, onSelect }: RowProps & { index: number }) {
  const { flag, title } = parseServerName(node.name)
  return (
    <button
      className={styles.row}
      aria-pressed={selected}
      disabled={disabled}
      onClick={onSelect}
      title={node.name}
      style={{ '--delay': `${Math.min(index, STAGGER_LIMIT) * STAGGER_MS}ms` } as CSSProperties}
    >
      <span className={styles.flag}>{flag}</span>
      <span className={styles.name}>
        <span className={styles.title}>{title}</span>
        <Protocol value={node.protocol} className={styles.sub} />
      </span>
      <Latency delay={node.delay} pending={pending} />
      <SelectedMark selected={selected} />
    </button>
  )
}

function SelectedMark({ selected }: { selected: boolean }) {
  return (
    <span className={styles.mark} data-on={selected || undefined}>
      <Check size={11} strokeWidth={3.4} />
    </span>
  )
}
