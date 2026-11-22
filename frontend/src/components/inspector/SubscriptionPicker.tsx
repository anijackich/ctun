import { Check, ChevronsUpDown, FileCode } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { useT } from '../../i18n'
import { daysUntil } from '../../lib/format'
import type { Profile } from '../../lib/types'
import { useApp } from '../../store/app'
import { useViewedProfile } from '../../store/selectors'
import { MenuItem } from '../Menu'
import menu from '../Menu.module.css'
import { UsageRing } from '../UsageRing'
import styles from './Inspector.module.css'
import { useAddActions } from './useAddActions'

export const usageRatio = (p: Profile) =>
  p.usage.total ? (p.usage.upload + p.usage.download) / p.usage.total : undefined

export function ProfileIcon({ profile, size }: { profile: Profile; size: number }) {
  return profile.local ? (
    <FileCode size={size} className={styles.configIcon} />
  ) : (
    <UsageRing ratio={usageRatio(profile)} />
  )
}

export function SubscriptionPicker({ onAddSubscription }: { onAddSubscription: () => void }) {
  const t = useT()
  const profiles = useApp((s) => s.profiles)
  const activeId = useApp((s) => s.activeProfileId)
  const running = useApp((s) => s.status === 'running')
  const viewed = useViewedProfile()
  const view = useApp((s) => s.viewProfile)
  const addActions = useAddActions(onAddSubscription)
  const sections = [
    { label: t.inspector.subscriptions, items: profiles.filter((p) => !p.local) },
    { label: t.inspector.configs, items: profiles.filter((p) => p.local) },
  ].filter((section) => section.items.length > 0)

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className={styles.picker} aria-label={t.inspector.choose}>
          {viewed && <ProfileIcon profile={viewed} size={16} />}
          <span className={styles.pickerText}>
            <span className={styles.pickerName}>{viewed?.name ?? t.inspector.choose}</span>
            {viewed && (
              <span className={styles.pickerHost}>{viewed.local ? t.inspector.localConfig : viewed.host}</span>
            )}
          </span>
          <ChevronsUpDown size={14} className={styles.chevron} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={`${menu.content} ${styles.menu}`} align="start" sideOffset={6}>
          {sections.map((section) => (
            <DropdownMenu.Group key={section.label}>
              <DropdownMenu.Label className={menu.label}>{section.label}</DropdownMenu.Label>
              {section.items.map((p) => {
                const days = p.usage.expire ? daysUntil(p.usage.expire) : undefined
                return (
                  <DropdownMenu.Item key={p.id} className={menu.item} onSelect={() => view(p.id)}>
                    <span className={menu.check}>{p.id === viewed?.id && <Check size={13} strokeWidth={3} />}</span>
                    <ProfileIcon profile={p} size={15} />
                    <span className={menu.grow}>{p.name}</span>
                    {p.id === activeId && (
                      <span className={styles.inUse} data-running={running || undefined} title={t.common.inUse} />
                    )}
                    {days !== undefined && <span className={menu.hint}>{t.subscription.expiresIn(days)}</span>}
                  </DropdownMenu.Item>
                )
              })}
            </DropdownMenu.Group>
          ))}
          {sections.length > 0 && <DropdownMenu.Separator className={menu.separator} />}
          {addActions.map(({ icon: Icon, label, hint, run }) => (
            <MenuItem key={label} icon={<Icon size={13} strokeWidth={2.4} />} hint={hint} onSelect={run}>
              {label}
            </MenuItem>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
