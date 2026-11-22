import { DropdownMenu } from 'radix-ui'
import type { ReactNode } from 'react'
import menu from './Menu.module.css'

interface MenuItemProps {
  icon?: ReactNode
  hint?: string
  destructive?: boolean
  disabled?: boolean
  onSelect: () => void
  children: ReactNode
}

export function MenuItem({ icon, hint, destructive, disabled, onSelect, children }: MenuItemProps) {
  return (
    <DropdownMenu.Item
      className={menu.item}
      data-destructive={destructive || undefined}
      disabled={disabled}
      onSelect={onSelect}
    >
      {icon !== undefined && <span className={menu.check}>{icon}</span>}
      <span className={menu.grow}>{children}</span>
      {hint && <span className={menu.hint}>{hint}</span>}
    </DropdownMenu.Item>
  )
}
