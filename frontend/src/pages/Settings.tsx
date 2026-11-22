import { System } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import { Button } from '../components/Button'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { FormRow, FormSection } from '../components/Form'
import { PageHeader } from '../components/PageHeader'
import { Segmented } from '../components/Segmented'
import { Switch } from '../components/Switch'
import { LOCALES, useT, type Locale } from '../i18n'
import { backend } from '../lib/backend'
import { LOG_LEVELS, THEMES, TRAY_ICONS, TUN_STACKS, type AppSettings } from '../lib/types'
import { useApp } from '../store/app'
import { DnsSection } from './DnsSection'
import styles from './Settings.module.css'

const PORT_RANGE = { min: 1024, max: 65535 } as const
const asOptions = <T extends string>(values: readonly T[]) =>
  values.map((value) => ({ value, label: value }))

export function Settings() {
  const t = useT()
  const locale = useApp((s) => s.locale)
  const settings = useApp((s) => s.settings)
  const setLocale = useApp((s) => s.setLocale)
  const updateSettings = useApp((s) => s.updateSettings)

  if (!settings) return null

  return (
    <>
      <PageHeader title={t.settings.title} />
      <div className={`${styles.content} enter`}>
        <FormSection title={t.settings.general}>
          <AppVersionRow />
          <FormRow label={t.settings.language}>
            <Segmented
              aria-label={t.settings.language}
              value={locale}
              options={(Object.keys(LOCALES) as Locale[]).map((value) => ({
                value,
                label: LOCALES[value],
              }))}
              onChange={setLocale}
            />
          </FormRow>
          {System.IsMac() && (
            <>
              <FormRow label={t.settings.appearance}>
                <Segmented
                  aria-label={t.settings.appearance}
                  value={settings.theme}
                  options={THEMES.map((value) => ({ value, label: t.settings.themes[value] }))}
                  onChange={(theme) => updateSettings({ theme })}
                />
              </FormRow>
              <FormRow label={t.settings.trayIcon} hint={t.settings.trayIconHint}>
                <Segmented
                  aria-label={t.settings.trayIcon}
                  value={settings.trayIcon}
                  options={TRAY_ICONS.map((value) => ({ value, label: t.settings.trayIcons[value] }))}
                  onChange={(trayIcon) => updateSettings({ trayIcon })}
                />
              </FormRow>
            </>
          )}
        </FormSection>

        <FormSection title={t.settings.core}>
          <CoreVersionRow />
          <FormRow label={t.settings.logLevel}>
            <Segmented
              aria-label={t.settings.logLevel}
              value={settings.logLevel}
              options={asOptions(LOG_LEVELS)}
              onChange={(logLevel) => updateSettings({ logLevel })}
            />
          </FormRow>
        </FormSection>

        <FormSection title={t.settings.proxy} footer={t.settings.proxyFooter}>
          <FormRow label={t.settings.port}>
            <PortInput
              key={settings.mixedPort}
              value={settings.mixedPort}
              onCommit={(mixedPort) => updateSettings({ mixedPort })}
            />
          </FormRow>
          <FormRow label={t.settings.password} hint={t.settings.passwordHint}>
            <Switch
              checked={settings.portAuth}
              onCheckedChange={(portAuth) => updateSettings({ portAuth })}
              aria-label={t.settings.password}
            />
          </FormRow>
          <SystemProxyRow settings={settings} />
        </FormSection>

        <FormSection title={t.settings.tun}>
          <FormRow label={t.settings.stack}>
            <Segmented
              aria-label={t.settings.stack}
              value={settings.tunStack}
              options={asOptions(TUN_STACKS)}
              onChange={(tunStack) => updateSettings({ tunStack })}
            />
          </FormRow>
          <HelperRow />
        </FormSection>

        <DnsSection dns={settings.dns} />
      </div>
    </>
  )
}

function PortInput({ value, onCommit }: { value: number; onCommit: (port: number) => void }) {
  const [draft, setDraft] = useState(String(value))

  const commit = () => {
    const port = Number(draft)
    if (Number.isInteger(port) && port >= PORT_RANGE.min && port <= PORT_RANGE.max) {
      if (port !== value) onCommit(port)
    } else {
      setDraft(String(value))
    }
  }

  return (
    <input
      type="number"
      className="mono"
      min={PORT_RANGE.min}
      max={PORT_RANGE.max}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
    />
  )
}

function AppVersionRow() {
  const t = useT()
  const [version, setVersion] = useState('')

  useEffect(() => {
    void backend.appVersion().then(setVersion, () => setVersion(''))
  }, [])

  return (
    <FormRow label={t.settings.version}>
      <span className="selectable mono">{version}</span>
    </FormRow>
  )
}

function CoreVersionRow() {
  const t = useT()
  const status = useApp((s) => s.status)
  const [version, setVersion] = useState<string>()

  useEffect(() => {
    if (status === 'starting' || status === 'stopping') return
    void backend.coreVersion().then(setVersion, () => setVersion(''))
  }, [status])

  return (
    <FormRow label="mihomo" hint={t.settings.coreHint}>
      <span className="selectable mono">{version === undefined ? '' : version || t.settings.coreMissing}</span>
    </FormRow>
  )
}

function SystemProxyRow({ settings }: { settings: AppSettings }) {
  const t = useT()
  const updateSettings = useApp((s) => s.updateSettings)
  const [supported, setSupported] = useState(true)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    void backend.systemProxySupported().then(setSupported, () => setSupported(false))
  }, [])

  const hint = !supported
    ? t.settings.systemProxyUnsupported
    : settings.tunnelMode === 'tun'
      ? t.settings.systemProxyTun
      : t.settings.systemProxyHint

  return (
    <FormRow label={t.settings.systemProxy} hint={hint}>
      <Switch
        checked={supported && settings.systemProxy}
        disabled={!supported}
        // Turning it on drops the port password, which exposes the exit IP: ask first.
        onCheckedChange={(on) =>
          on && settings.portAuth ? setConfirming(true) : updateSettings({ systemProxy: on })
        }
        aria-label={t.settings.systemProxy}
      />
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={t.settings.systemProxyConfirmTitle}
        body={t.settings.systemProxyConfirmBody}
        confirm={t.settings.systemProxyConfirm}
        onConfirm={() => updateSettings({ systemProxy: true })}
      />
    </FormRow>
  )
}

function HelperRow() {
  const t = useT()
  const helper = useApp((s) => s.helper)
  const busy = useApp((s) => s.helperBusy)
  const installHelper = useApp((s) => s.installHelper)
  const uninstallHelper = useApp((s) => s.uninstallHelper)
  const error = useApp((s) => s.error)
  const failed = error && (!error.code || error.code === 'helper-missing') ? error.message : undefined

  return (
    <FormRow label={t.settings.helper} hint={failed ?? t.settings.helperHint}>
      {helper && <span data-state={helper}>{t.settings.helperState[helper]}</span>}
      {helper === 'installed' ? (
        <Button disabled={busy} onClick={() => void uninstallHelper()}>
          {t.settings.uninstall}
        </Button>
      ) : (
        helper !== 'unsupported' && (
          <Button variant="primary" disabled={busy} onClick={() => void installHelper()}>
            {busy ? t.settings.installing : helper === 'outdated' ? t.settings.reinstall : t.settings.install}
          </Button>
        )
      )}
    </FormRow>
  )
}
