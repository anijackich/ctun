import type { ReactNode } from 'react'
import styles from './PageHeader.module.css'

export function PageHeader({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <header className={styles.header}>
      <h1 className={styles.title}>{title}</h1>
      {children && <div className={styles.actions}>{children}</div>}
    </header>
  )
}
