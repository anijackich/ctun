import type { ButtonHTMLAttributes } from 'react'
import styles from './Button.module.css'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'primary' | 'plain' | 'danger'
  icon?: boolean
}

export function Button({ variant = 'default', icon, className, ...rest }: ButtonProps) {
  const classes = [styles.button, styles[variant], icon && styles.icon, className]
  return <button type="button" className={classes.filter(Boolean).join(' ')} {...rest} />
}
