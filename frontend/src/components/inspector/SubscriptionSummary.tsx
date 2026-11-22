import { Browser } from '@wailsio/runtime'
import { Ellipsis, Info, RefreshCw } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { useMemo, useState } from 'react'
import { useFormat, useT } from '../../i18n'
import { profiles } from '../../lib/backend'
import { copyText } from '../../lib/clipboard'
import { errorMessage } from '../../lib/errors'
import { daysUntil } from '../../lib/format'
import { isServer } from '../../lib/groups'
import type { Profile } from '../../lib/types'
import { useApp } from '../../store/app'
import { Button } from '../Button'
import { ConfirmDialog } from '../ConfirmDialog'
import { MenuItem } from '../Menu'
import menu from '../Menu.module.css'
import { ProfileSheet } from '../ProfileSheet'
import styles from './Inspector.module.css'
import { usageRatio } from './SubscriptionPicker'

export function SubscriptionSummary({ profile }: { profile: Profile }) {
  const t = useT()
  const format = useFormat()
  const inUse = useApp((s) => s.activeProfileId === profile.id)
  const running = useApp((s) => s.status === 'running')
  const groups = useApp((s) => s.groups[profile.id])
  const serverCount = useMemo(
    () => new Set((groups ?? []).flatMap((g) => g.nodes.filter(isServer).map((n) => n.name))).size,
    [groups],
  )
  const refreshProfile = useApp((s) => s.refreshProfile)
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState<string>()
  const { usage } = profile
  const used = usage.upload + usage.download
  const ratio = usageRatio(profile)
  const days = usage.expire ? daysUntil(usage.expire) : undefined

  const refresh = async () => {
    setRefreshing(true)
    setRefreshError(undefined)
    try {
      await refreshProfile(profile.id)
    } catch (err) {
      setRefreshError(errorMessage(err))
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <section className={styles.summary}>
      <div className={styles.summaryTop}>
        <span className="mono tabular">
          {profile.local
            ? t.inspector.servers(serverCount)
            : usage.total
              ? t.subscription.used(format.bytes(used), format.bytes(usage.total))
              : t.subscription.usedUnlimited(format.bytes(used))}
        </span>
        <span className={styles.actions}>
          {profile.announce && (
            <span className={styles.announce} title={profile.announce} role="note" aria-label={profile.announce}>
              <Info size={13} />
            </span>
          )}
          {!profile.local && (
            <Button
              variant="plain"
              icon
              data-fx="refresh"
              data-busy={refreshing || undefined}
              className={styles.refresh}
              disabled={refreshing}
              onClick={() => void refresh()}
              aria-label={t.subscription.refresh}
              title={t.subscription.refresh}
            >
              <RefreshCw size={13} />
            </Button>
          )}
          <SubscriptionMenu profile={profile} onRefresh={() => void refresh()} />
        </span>
      </div>
      {ratio !== undefined && (
        <span className={styles.bar}>
          <span style={{ transform: `scaleX(${Math.min(1, ratio)})` }} />
        </span>
      )}
      <div className={styles.usageState} data-in-use={inUse || undefined}>
        <span className={styles.inUse} data-running={(inUse && running) || undefined} />
        {inUse ? t.common.inUse : t.inspector.notInUse}
      </div>
      <div className={styles.meta}>
        {days !== undefined && (
          <span data-warn={days <= 7 || undefined}>{days > 0 ? t.subscription.expiresIn(days) : t.subscription.expired}</span>
        )}
        <span>
          {profile.local
            ? t.inspector.imported(format.relative(profile.updatedAt))
            : t.subscription.updated(format.relative(profile.updatedAt))}
        </span>
      </div>
      {refreshError && (
        <div className={styles.refreshError} role="alert">
          {refreshError}
        </div>
      )}
    </section>
  )
}

function SubscriptionMenu({ profile, onRefresh }: { profile: Profile; onRefresh: () => void }) {
  const t = useT()
  const removeProfile = useApp((s) => s.removeProfile)
  const setNotice = useApp((s) => s.setNotice)
  const [confirming, setConfirming] = useState(false)
  const [editing, setEditing] = useState(false)
  const site = profile.webPageUrl ?? profile.supportUrl
  const showError = (err: unknown) => setNotice(errorMessage(err))

  return (
    <>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <Button variant="plain" icon data-fx="dots" data-fx-on="press" aria-label={t.subscription.actions}>
            <Ellipsis size={15} />
          </Button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className={menu.content} align="end" sideOffset={4}>
            {!profile.local && (
              <>
                <MenuItem onSelect={onRefresh}>{t.subscription.menu.refresh}</MenuItem>
                <MenuItem onSelect={() => void profiles.link(profile.id).then(copyText).catch(showError)}>
                  {t.subscription.menu.copyLink}
                </MenuItem>
                {site && <MenuItem onSelect={() => void Browser.OpenURL(site)}>{t.subscription.menu.openSite}</MenuItem>}
              </>
            )}
            <MenuItem onSelect={() => setEditing(true)}>
              {profile.local ? t.subscription.menu.rename : t.subscription.menu.edit}
            </MenuItem>
            <DropdownMenu.Separator className={menu.separator} />
            <MenuItem destructive onSelect={() => setConfirming(true)}>
              {t.subscription.menu.remove}
            </MenuItem>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={t.subscription.removeTitle(profile.name)}
        body={profile.local ? t.subscription.removeConfigBody : t.subscription.removeBody}
        confirm={t.subscription.menu.remove}
        onConfirm={() => void removeProfile(profile.id).catch(showError)}
      />
      {editing && <ProfileSheet profile={profile} onClose={() => setEditing(false)} />}
    </>
  )
}
