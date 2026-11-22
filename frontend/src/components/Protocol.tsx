import { Fragment } from 'react'
import styles from './Protocol.module.css'

export function Protocol({ value, className }: { value?: string; className?: string }) {
  if (!value) return null
  const parts = value.split('·').map((part) => part.trim())
  return (
    <span className={`${styles.protocol} mono ${className ?? ''}`}>
      {parts.map((part, i) => (
        <Fragment key={i}>
          {i > 0 && <span className={styles.dot}>·</span>}
          {part}
        </Fragment>
      ))}
    </span>
  )
}
