import { useState } from 'react'
import styles from './Sheet.module.css'
import { SheetForm } from './SheetForm'

interface NameDialogProps {
  title: string
  label: string
  placeholder: string
  initial: string
  submit: string
  onSubmit: (name: string) => Promise<void>
  onClose: () => void
}

export function NameDialog({ title, label, placeholder, initial, submit, onSubmit, onClose }: NameDialogProps) {
  const [name, setName] = useState(initial)
  const trimmed = name.trim()
  return (
    <SheetForm title={title} submit={submit} canSubmit={trimmed !== ''} onSubmit={() => onSubmit(trimmed)} onClose={onClose}>
      <label className={styles.field}>
        <span>{label}</span>
        <input autoFocus placeholder={placeholder} value={name} onChange={(e) => setName(e.target.value)} />
      </label>
    </SheetForm>
  )
}
