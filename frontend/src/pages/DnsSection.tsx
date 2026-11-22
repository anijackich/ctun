import { useState } from 'react'
import { Button } from '../components/Button'
import { FormRow, FormSection } from '../components/Form'
import { Switch } from '../components/Switch'
import { useT } from '../i18n'
import { backend } from '../lib/backend'
import { errorMessage } from '../lib/errors'
import { DNS_LISTS, type DnsList, type DnsSettings } from '../lib/types'
import { useApp } from '../store/app'
import styles from './DnsSection.module.css'

export function DnsSection({ dns }: { dns: DnsSettings }) {
  const t = useT()
  const updateSettings = useApp((s) => s.updateSettings)
  const setNotice = useApp((s) => s.setNotice)
  const update = (patch: Partial<DnsSettings>) => updateSettings({ dns: { ...dns, ...patch } })

  return (
    <FormSection title={t.dns.title} footer={t.dns.footer}>
      {DNS_LISTS.map((list) => (
        // Remount when the saved list changes, e.g. after a reset or normalization.
        <NameserverField
          key={`${list}:${dns[list].join()}`}
          list={list}
          value={dns[list]}
          onCommit={(entries) => update({ [list]: entries })}
        />
      ))}
      <FormRow label={t.dns.viaProxy} hint={t.dns.viaProxyHint}>
        <Switch checked={dns.viaProxy} onCheckedChange={(viaProxy) => update({ viaProxy })} aria-label={t.dns.viaProxy} />
      </FormRow>
      <FormRow label={t.dns.systemOverride} hint={t.dns.systemOverrideHint}>
        <Switch
          checked={dns.systemOverride}
          onCheckedChange={(systemOverride) => update({ systemOverride })}
          aria-label={t.dns.systemOverride}
        />
      </FormRow>
      <FormRow label={t.dns.defaults} hint={t.dns.defaultsHint}>
        <Button onClick={() => void backend.resetDNS().catch((err: unknown) => setNotice(errorMessage(err)))}>
          {t.dns.reset}
        </Button>
      </FormRow>
    </FormSection>
  )
}

interface NameserverFieldProps {
  list: DnsList
  value: string[]
  onCommit: (entries: string[]) => void
}

function NameserverField({ list, value, onCommit }: NameserverFieldProps) {
  const t = useT()
  const [draft, setDraft] = useState(value.join('\n'))
  const [invalid, setInvalid] = useState<number[]>([])
  const { label, hint } = t.dns.lists[list]

  const commit = async () => {
    const lines = draft.split('\n')
    const bad = await backend.invalidNameservers(lines)
    setInvalid(bad)
    const entries = lines.map((line) => line.trim()).filter(Boolean)
    if (bad.length === 0 && entries.join('\n') !== value.join('\n')) onCommit(entries)
  }

  return (
    <div className={styles.field}>
      <div className={styles.label}>
        <span>{label}</span>
        <span className={styles.hint}>{hint}</span>
      </div>
      <textarea
        className={`${styles.entries} mono`}
        spellCheck={false}
        autoCapitalize="off"
        autoCorrect="off"
        rows={Math.max(2, draft.split('\n').length)}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => void commit()}
        aria-label={label}
        aria-invalid={invalid.length > 0 || undefined}
      />
      {invalid.length > 0 && (
        <ul className={styles.errors}>
          {invalid.map((line) => (
            <li key={line}>{t.dns.invalid(line)}</li>
          ))}
        </ul>
      )}
    </div>
  )
}
