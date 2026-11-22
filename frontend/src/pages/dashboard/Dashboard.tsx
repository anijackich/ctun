import { ArrowDown, ArrowUp, PanelRight } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from '../../components/Button'
import { Card } from '../../components/Card'
import { CopyButton } from '../../components/CopyButton'
import { Latency } from '../../components/Latency'
import { PageHeader } from '../../components/PageHeader'
import { Protocol } from '../../components/Protocol'
import { Sparkline } from '../../components/Sparkline'
import { useFormat, useT } from '../../i18n'
import { backend } from '../../lib/backend'
import { countryFlag, GENERIC_FLAG, parseServerName } from '../../lib/flag'
import { useApp } from '../../store/app'
import { TRAFFIC_HISTORY } from '../../store/core'
import { useActiveNode, useTunnelMode } from '../../store/selectors'
import { ConnectionCard } from './ConnectionCard'
import styles from './Dashboard.module.css'

export function Dashboard() {
  const t = useT()
  const inspectorOpen = useApp((s) => s.inspectorOpen)
  const toggleInspector = useApp((s) => s.toggleInspector)

  return (
    <>
      <PageHeader title={t.dashboard.title}>
        <Button
          variant="plain"
          icon
          data-fx="panel"
          aria-pressed={inspectorOpen}
          aria-label={t.inspector.toggle}
          title={t.inspector.toggle}
          onClick={toggleInspector}
        >
          <PanelRight size={16} />
        </Button>
      </PageHeader>
      <div className={styles.content}>
        <div className="enter">
          <ConnectionCard />
        </div>
        <div className={styles.grid}>
          <div className={`${styles.wide} enter`} style={{ animationDelay: '60ms' }}>
            <TrafficCard />
          </div>
          <div className="enter" style={{ animationDelay: '120ms' }}>
            <ExitNodeCard />
          </div>
          <div className="enter" style={{ animationDelay: '160ms' }}>
            <SessionCard />
          </div>
        </div>
      </div>
    </>
  )
}

function TrafficCard() {
  const t = useT()
  const format = useFormat()
  const traffic = useApp((s) => s.traffic)
  const session = useApp((s) => s.session)
  const running = useApp((s) => s.status === 'running')
  const latest = traffic.at(-1)

  return (
    <Card title={t.dashboard.traffic}>
      <div className={styles.speeds}>
        <Metric icon={<ArrowDown size={14} className={styles.down} />} value={format.speed(latest?.down ?? 0)} large />
        <Metric icon={<ArrowUp size={14} className={styles.up} />} value={format.speed(latest?.up ?? 0)} large />
        <div className={styles.spacer} />
        <Metric label={t.dashboard.downloaded} value={format.bytes(session.downloaded)} />
        <Metric label={t.dashboard.uploaded} value={format.bytes(session.uploaded)} />
      </div>
      <div className={styles.chart}>
        <Sparkline points={traffic} capacity={TRAFFIC_HISTORY} height={110} />
        {!running && <span className={styles.idle}>{t.dashboard.idle}</span>}
      </div>
    </Card>
  )
}

function ExitNodeCard() {
  const t = useT()
  const node = useActiveNode()
  const exit = useApp((s) => s.exit)
  const running = useApp((s) => s.status === 'running')
  const { flag, title: name } = node ? parseServerName(node.name) : { flag: GENERIC_FLAG, title: '—' }

  return (
    <Card title={t.dashboard.exitNode}>
      <div className={styles.node}>
        <span className={styles.nodeFlag}>{flag}</span>
        <span className={styles.nodeText}>
          <span className={styles.nodeName}>{name}</span>
          <Protocol value={node?.protocol} className={styles.muted} />
        </span>
      </div>
      <dl className={styles.facts}>
        <dt>{t.dashboard.latency}</dt>
        <dd>
          <Latency delay={node?.delay} />
        </dd>
        <dt>{t.dashboard.exitIp}</dt>
        <dd className={styles.exit}>
          {exit ? (
            <>
              <span className="mono selectable">{exit.ip}</span>
              <span className={styles.muted} title={exit.org}>
                {[countryFlag(exit.country), exit.city || exit.country].filter(Boolean).join(' ')}
              </span>
            </>
          ) : (
            <span className={styles.muted}>{running ? t.dashboard.exitLookup : '—'}</span>
          )}
        </dd>
      </dl>
    </Card>
  )
}

function SessionCard() {
  const t = useT()
  const format = useFormat()
  const session = useApp((s) => s.session)
  const running = useApp((s) => s.status === 'running')
  const tunnelMode = useTunnelMode()
  const port = useApp((s) => s.settings?.mixedPort)

  return (
    <Card title={t.connection.mode}>
      <div className={styles.modeBadge}>{t.connection.tunnel[tunnelMode]}</div>
      <dl className={styles.facts}>
        {tunnelMode === 'proxy' && (
          <>
            <dt>{t.dashboard.proxyAddress}</dt>
            <dd className={styles.proxyAddress}>
              <span className="mono selectable">127.0.0.1:{port}</span>
              {running && <CopyButton text={backend.proxyURL} label={t.dashboard.copyProxy} />}
            </dd>
          </>
        )}
        <dt>{t.dashboard.connections}</dt>
        <dd className="mono tabular">{running ? session.connections : '—'}</dd>
        <dt>{t.dashboard.memory}</dt>
        <dd className="mono tabular">{running ? format.bytes(session.memory) : '—'}</dd>
      </dl>
    </Card>
  )
}

interface MetricProps {
  icon?: ReactNode
  label?: string
  value: string
  large?: boolean
}

function Metric({ icon, label, value, large }: MetricProps) {
  return (
    <span className={styles.metric} data-large={large || undefined}>
      {label && <span className={styles.metricLabel}>{label}</span>}
      <span className="mono tabular">
        {icon}
        {value}
      </span>
    </span>
  )
}
