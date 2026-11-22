import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useT } from '../../i18n'
import { readText } from '../../lib/clipboard'
import { isSubscriptionUrl } from '../../lib/url'
import { useViewedProfile } from '../../store/selectors'
import { Button } from '../Button'
import { ProfileSheet } from '../ProfileSheet'
import { ServerList } from '../ServerList'
import styles from './Inspector.module.css'
import { SubscriptionPicker } from './SubscriptionPicker'
import { SubscriptionSummary } from './SubscriptionSummary'
import { useAddActions } from './useAddActions'

export function Inspector() {
  const t = useT()
  const profile = useViewedProfile()
  const [draftUrl, setDraftUrl] = useState<string>()

  const openSubscription = async () => {
    const clip = await readText()
    setDraftUrl(isSubscriptionUrl(clip) ? clip.trim() : '')
  }

  return (
    <aside className={styles.inspector}>
      <header className={styles.header}>
        <SubscriptionPicker onAddSubscription={() => void openSubscription()} />
        <button
          className={styles.add}
          data-fx="plus"
          onClick={() => void openSubscription()}
          aria-label={t.empty.action}
          title={t.empty.action}
        >
          <Plus size={15} strokeWidth={2.4} />
        </button>
      </header>
      {profile ? (
        <div className={styles.body} key={profile.id}>
          <div className="enter">
            <SubscriptionSummary profile={profile} />
          </div>
          <div className={`${styles.servers} enter`} style={{ animationDelay: '60ms' }}>
            <ServerList profileId={profile.id} />
          </div>
        </div>
      ) : (
        <EmptyState onAddSubscription={() => void openSubscription()} />
      )}
      {draftUrl !== undefined && <ProfileSheet initialUrl={draftUrl} onClose={() => setDraftUrl(undefined)} />}
    </aside>
  )
}

function EmptyState({ onAddSubscription }: { onAddSubscription: () => void }) {
  const t = useT()
  const [subscription, ...imports] = useAddActions(onAddSubscription)
  return (
    <div className={`${styles.emptyState} enter`}>
      <h2>{t.empty.title}</h2>
      <p>{t.empty.body}</p>
      <Button variant="primary" onClick={subscription.run}>
        <Plus size={14} strokeWidth={2.5} />
        {t.empty.action}
      </Button>
      <div className={styles.emptyActions}>
        {imports.map(({ icon: Icon, label, short, run }) => (
          <Button key={label} onClick={run}>
            <Icon size={14} />
            {short ?? label}
          </Button>
        ))}
      </div>
    </div>
  )
}
