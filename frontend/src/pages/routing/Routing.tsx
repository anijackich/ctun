import { useState } from 'react'
import { PageHeader } from '../../components/PageHeader'
import { SearchField } from '../../components/SearchField'
import { Segmented } from '../../components/Segmented'
import { useT } from '../../i18n'
import { ActiveRules } from './ActiveRules'
import styles from './Routing.module.css'
import { RoutingProfiles } from './RoutingProfiles'

type Tab = 'profile' | 'rules'

export function Routing() {
  const t = useT()
  const [tab, setTab] = useState<Tab>('profile')
  const [query, setQuery] = useState('')

  return (
    <>
      <PageHeader title={t.nav.routing}>
        {tab === 'rules' && (
          <SearchField className={styles.search} placeholder={t.routing.search} value={query} onChange={setQuery} />
        )}
        <Segmented
          aria-label={t.nav.routing}
          value={tab}
          options={(['profile', 'rules'] as const).map((value) => ({ value, label: t.routingProfile.tabs[value] }))}
          onChange={setTab}
        />
      </PageHeader>
      {tab === 'profile' ? (
        <div className={styles.content}>
          <RoutingProfiles />
        </div>
      ) : (
        <ActiveRules query={query} />
      )}
    </>
  )
}
