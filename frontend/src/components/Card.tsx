import type { HTMLAttributes } from 'react'
import styles from './Card.module.css'

export function Card({ title, className, children, ...rest }: HTMLAttributes<HTMLElement> & { title?: string }) {
  return (
    <section className={[styles.card, className].filter(Boolean).join(' ')} {...rest}>
      {title && (
        <header className={styles.header}>
          <h2 className={styles.title}>{title}</h2>
        </header>
      )}
      {children}
    </section>
  )
}
