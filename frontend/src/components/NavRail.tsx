import {
  ArrowLeftRight,
  Gauge,
  Route,
  ScrollText,
  Settings,
  type LucideIcon,
} from 'lucide-react'
import { useT } from '../i18n'
import type { Page } from '../lib/types'
import { useApp } from '../store/app'
import styles from './NavRail.module.css'

interface NavEntry {
  page: Page
  icon: LucideIcon
  fx: string
}

const MAIN: NavEntry[] = [
  { page: 'dashboard', icon: Gauge, fx: 'gauge' },
  { page: 'routing', icon: Route, fx: 'route' },
  { page: 'connections', icon: ArrowLeftRight, fx: 'arrows' },
  { page: 'logs', icon: ScrollText, fx: 'scroll' },
]

export function NavRail() {
  return (
    <aside className={styles.rail}>
      <div className={styles.titlebar} />
      <nav className={styles.nav}>
        {MAIN.map((item) => (
          <NavItem key={item.page} {...item} />
        ))}
      </nav>
      <div className={styles.spacer} />
      <nav className={styles.nav}>
        <NavItem page="settings" icon={Settings} fx="gear" />
      </nav>
    </aside>
  )
}

function NavItem({ page, icon: Icon, fx }: NavEntry) {
  const t = useT()
  const current = useApp((s) => s.page)
  const navigate = useApp((s) => s.navigate)
  const active = current === page

  return (
    <button
      className={styles.item}
      data-fx={fx}
      data-active={active || undefined}
      aria-current={active ? 'page' : undefined}
      onClick={() => navigate(page)}
    >
      <span className={styles.icon}>
        <Icon size={18} strokeWidth={1.9} />
      </span>
      <span className={styles.label}>{t.nav[page]}</span>
    </button>
  )
}
