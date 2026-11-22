import { Check, ChevronsUpDown, Copy, Pencil, Plus, Trash2 } from 'lucide-react'
import { DropdownMenu } from 'radix-ui'
import { useCallback, useEffect, useState } from 'react'
import { Button } from '../../components/Button'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { MenuItem } from '../../components/Menu'
import menu from '../../components/Menu.module.css'
import { NameDialog } from '../../components/NameDialog'
import { useT } from '../../i18n'
import { routing as routingApi } from '../../lib/backend'
import { errorMessage } from '../../lib/errors'
import type { RoutingProfileState } from '../../lib/types'
import { useApp } from '../../store/app'
import { ProfileEditor, SyntaxHelp } from './ProfileEditor'
import styles from './RoutingProfile.module.css'

type NameAction = 'create' | 'duplicate' | 'rename'

function useRoutingProfiles(showError: (err: unknown) => void) {
  const [profiles, setProfiles] = useState<RoutingProfileState[]>([])
  useEffect(() => {
    const off = routingApi.onProfiles(setProfiles)
    routingApi.profiles().then(setProfiles, showError)
    return off
  }, [showError])
  return profiles
}

export function RoutingProfiles() {
  const t = useT()
  const setNotice = useApp((s) => s.setNotice)
  const showError = useCallback((err: unknown) => setNotice(errorMessage(err)), [setNotice])
  const profiles = useRoutingProfiles(showError)
  const [viewedId, setViewedId] = useState<string>()
  const [naming, setNaming] = useState<NameAction>()
  const [removing, setRemoving] = useState(false)

  const active = profiles.find((p) => p.active)
  const viewed = profiles.find((p) => p.profile.id === viewedId) ?? active
  if (!viewed) return null

  const submitName = async (name: string) => {
    if (naming === 'rename') {
      await routingApi.saveProfile({ ...viewed.profile, name })
      return
    }
    const created = await routingApi.createProfile(name, naming === 'duplicate' ? viewed.profile.id : '')
    setViewedId(created.profile.id)
  }

  return (
    <div className={styles.profiles}>
      <div className={styles.bar}>
        <ProfilePicker
          profiles={profiles}
          viewed={viewed}
          onView={setViewedId}
          onName={setNaming}
          onRemove={() => setRemoving(true)}
        />
        {viewed.active ? (
          <span className={styles.inUse}>{t.common.inUse}</span>
        ) : (
          <Button variant="primary" onClick={() => void routingApi.activateProfile(viewed.profile.id).catch(showError)}>
            {t.routingProfile.use}
          </Button>
        )}
        <span className={styles.spacer} />
        <SyntaxHelp />
      </div>
      <ProfileEditor key={viewed.profile.id} state={viewed} />
      {naming && (
        <NameDialog
          title={t.routingProfile.nameTitle[naming]}
          label={t.routingProfile.name}
          placeholder={t.routingProfile.namePlaceholder}
          initial={naming === 'rename' ? viewed.profile.name : naming === 'duplicate' ? `${viewed.profile.name} 2` : ''}
          submit={naming === 'rename' ? t.common.save : t.routingProfile.create}
          onSubmit={submitName}
          onClose={() => setNaming(undefined)}
        />
      )}
      <ConfirmDialog
        open={removing}
        onOpenChange={setRemoving}
        title={t.routingProfile.removeTitle(viewed.profile.name)}
        body={t.routingProfile.removeBody}
        confirm={t.routingProfile.remove}
        onConfirm={() =>
          void routingApi.deleteProfile(viewed.profile.id).then(() => setViewedId(undefined), showError)
        }
      />
    </div>
  )
}

interface ProfilePickerProps {
  profiles: RoutingProfileState[]
  viewed: RoutingProfileState
  onView: (id: string) => void
  onName: (action: NameAction) => void
  onRemove: () => void
}

function ProfilePicker({ profiles, viewed, onView, onName, onRemove }: ProfilePickerProps) {
  const t = useT()
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button className={styles.picker} aria-label={t.routingProfile.choose}>
          <span className={styles.pickerName}>{viewed.profile.name}</span>
          <ChevronsUpDown size={13} className={styles.chevron} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={menu.content} align="start" sideOffset={6}>
          {profiles.map((p) => (
            <DropdownMenu.Item key={p.profile.id} className={menu.item} onSelect={() => onView(p.profile.id)}>
              <span className={menu.check}>{p.profile.id === viewed.profile.id && <Check size={13} strokeWidth={3} />}</span>
              <span className={menu.grow}>{p.profile.name}</span>
              {p.active && <span className={styles.activeDot} title={t.common.inUse} />}
            </DropdownMenu.Item>
          ))}
          <DropdownMenu.Separator className={menu.separator} />
          <MenuItem icon={<Plus size={13} strokeWidth={2.5} />} onSelect={() => onName('create')}>
            {t.routingProfile.newProfile}
          </MenuItem>
          <MenuItem icon={<Copy size={13} />} onSelect={() => onName('duplicate')}>
            {t.routingProfile.duplicate}
          </MenuItem>
          <MenuItem icon={<Pencil size={13} />} onSelect={() => onName('rename')}>
            {t.routingProfile.rename}
          </MenuItem>
          <MenuItem icon={<Trash2 size={13} />} destructive disabled={profiles.length === 1} onSelect={onRemove}>
            {t.routingProfile.remove}
          </MenuItem>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
