import { Button } from '../../components/Button'
import { Segmented } from '../../components/Segmented'
import { TunnelButton } from '../../components/TunnelButton'
import { useT } from '../../i18n'
import { formatUptime } from '../../lib/format'
import { ROUTING_MODES, TUNNEL_MODES, type CoreErrorCode, type Traffic } from '../../lib/types'
import { useNow } from '../../lib/useNow'
import { useApp } from '../../store/app'
import { useActiveNode, useActiveProfile, useRoutingMode, useTunnelMode } from '../../store/selectors'
import styles from './Dashboard.module.css'

const FULL_SCALE = Math.log10(1 + 20 * 1024)

const activityLevel = (traffic?: Traffic) =>
  traffic ? Math.min(1, Math.log10(1 + (traffic.down + traffic.up) / 1024) / FULL_SCALE) : 0

export function ConnectionCard() {
  const t = useT()
  const status = useApp((s) => s.status)
  const startedAt = useApp((s) => s.startedAt)
  const tunnelMode = useTunnelMode()
  const routingMode = useRoutingMode()
  const error = useApp((s) => s.error)
  const installing = useApp((s) => s.helperBusy)
  const shown = installing ? 'starting' : status
  const latest = useApp((s) => s.traffic.at(-1))
  const toggleConnection = useApp((s) => s.toggleConnection)
  const setTunnelMode = useApp((s) => s.setTunnelMode)
  const setRoutingMode = useApp((s) => s.setRoutingMode)
  const profile = useActiveProfile()
  const node = useActiveNode()
  const now = useNow(1000, status === 'running')
  const running = status === 'running'

  return (
    <section className={styles.connection} data-status={shown}>
      <div className={styles.aurora} aria-hidden />
      <TunnelButton
        status={shown}
        level={activityLevel(latest)}
        label={running ? t.connection.disconnect : t.connection.connect}
        onToggle={toggleConnection}
      />
      <div className={styles.connectionBody}>
        <div className={styles.statusLine}>
          <span className={styles.statusTitle}>{t.status[shown]}</span>
          {running && startedAt && (
            <span className={`${styles.uptime} mono tabular`}>{formatUptime(now - startedAt)}</span>
          )}
        </div>
        <div className={styles.via}>
          {node ? t.connection.via(node.name) : ' '}
          {profile && profile.name !== node?.name && (
            <span className={styles.viaProfile}> · {profile.name}</span>
          )}
        </div>
        {error && <CoreErrorLine code={error.code} message={error.message} />}
        <div className={styles.controls}>
          <Segmented
            aria-label={t.connection.mode}
            value={tunnelMode}
            options={TUNNEL_MODES.map((value) => ({ value, label: t.connection.tunnel[value] }))}
            onChange={setTunnelMode}
          />
          <Segmented
            aria-label={t.connection.routing}
            value={routingMode}
            options={ROUTING_MODES.map((value) => ({ value, label: t.connection.routes[value] }))}
            onChange={setRoutingMode}
          />
        </div>
      </div>
    </section>
  )
}

function CoreErrorLine({ code, message }: { code?: CoreErrorCode; message: string }) {
  const t = useT()
  const busy = useApp((s) => s.helperBusy)
  const installHelper = useApp((s) => s.installHelper)
  const needsHelper = code === 'helper-missing' || code === 'helper-outdated'
  return (
    <div className={styles.error} role="alert" title={message}>
      <span>{code ? t.errors[code] : message}</span>
      {needsHelper && (
        <Button
          variant="primary"
          disabled={busy}
          onClick={() => void installHelper()}
        >
          {busy ? t.settings.installing : t.settings.installHelper}
        </Button>
      )}
    </div>
  )
}
