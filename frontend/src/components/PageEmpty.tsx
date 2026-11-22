import type { ReactNode } from 'react'
import styles from './PageEmpty.module.css'

export function PageEmpty({ children }: { children?: ReactNode }) {
  return <p className={styles.empty}>{children}</p>
}
