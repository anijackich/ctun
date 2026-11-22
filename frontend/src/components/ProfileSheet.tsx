import { useState } from 'react'
import { useT } from '../i18n'
import type { Profile } from '../lib/types'
import { isSubscriptionUrl } from '../lib/url'
import { useApp } from '../store/app'
import styles from './Sheet.module.css'
import { SheetForm } from './SheetForm'

const USER_AGENTS = [
  { label: 'mihomo', value: 'mihomo/v1.19.34' },
  { label: 'clash-verge', value: 'clash-verge/v2.2.3' },
  { label: 'v2rayN', value: 'v2rayN/7.0' },
  { label: 'Happ', value: 'Happ/2.0.0' },
]
const DEFAULT_USER_AGENT = USER_AGENTS[0].value

const PROVIDER_INTERVAL = 0
const MANUAL_INTERVAL = -1
const HOURLY_INTERVALS = [1, 6, 12, 24]

interface ProfileSheetProps {
  initialUrl?: string
  profile?: Profile
  onClose: () => void
}

export function ProfileSheet({ initialUrl = '', profile, onClose }: ProfileSheetProps) {
  const t = useT()
  const addProfile = useApp((s) => s.addProfile)
  const editProfile = useApp((s) => s.editProfile)
  const [url, setUrl] = useState(profile ? '' : initialUrl)
  const [name, setName] = useState(profile?.name ?? '')
  const [userAgent, setUserAgent] = useState(profile?.userAgent || DEFAULT_USER_AGENT)
  const [updateInterval, setUpdateInterval] = useState(profile?.updateIntervalHours ?? PROVIDER_INTERVAL)

  const editing = profile !== undefined
  const subscription = !profile?.local
  const link = url.trim()
  const valid = editing ? name.trim() !== '' && (!link || isSubscriptionUrl(link)) : isSubscriptionUrl(link)
  const userAgents = USER_AGENTS.some((ua) => ua.value === userAgent)
    ? USER_AGENTS
    : [...USER_AGENTS, { label: userAgent, value: userAgent }]

  const submit = () => {
    const fields = { url: link, name: name.trim(), userAgent, updateIntervalHours: updateInterval }
    return profile ? editProfile({ id: profile.id, ...fields }) : addProfile(fields)
  }

  return (
    <SheetForm
      title={!editing ? t.addSheet.title : subscription ? t.addSheet.editTitle : t.addSheet.editConfigTitle}
      submit={editing ? t.common.save : t.addSheet.add}
      pendingLabel={t.common.checking}
      canSubmit={valid}
      hint={subscription && (editing ? t.addSheet.editHint : t.addSheet.hint)}
      onSubmit={submit}
      onClose={onClose}
    >
      {subscription && (
        <label className={styles.field}>
          <span>{t.addSheet.url}</span>
          <input
            autoFocus={!editing}
            type="url"
            spellCheck={false}
            placeholder={editing ? t.addSheet.keepUrl : 'https://…'}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
          />
        </label>
      )}
      <label className={styles.field}>
        <span>{t.addSheet.name}</span>
        <input
          autoFocus={editing}
          placeholder={t.addSheet.namePlaceholder}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      {subscription && (
        <>
          <label className={styles.field}>
            <span>{t.addSheet.userAgent}</span>
            <select value={userAgent} onChange={(e) => setUserAgent(e.target.value)}>
              {userAgents.map((ua) => (
                <option key={ua.value} value={ua.value}>
                  {ua.label}
                </option>
              ))}
            </select>
          </label>
          <label className={styles.field}>
            <span>{t.addSheet.update}</span>
            <select value={updateInterval} onChange={(e) => setUpdateInterval(Number(e.target.value))}>
              <option value={PROVIDER_INTERVAL}>{t.addSheet.intervals.provider}</option>
              {HOURLY_INTERVALS.map((hours) => (
                <option key={hours} value={hours}>
                  {t.addSheet.intervals.every(hours)}
                </option>
              ))}
              <option value={MANUAL_INTERVAL}>{t.addSheet.intervals.manual}</option>
            </select>
          </label>
        </>
      )}
    </SheetForm>
  )
}
