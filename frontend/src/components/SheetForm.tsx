import { Dialog } from 'radix-ui'
import { useState, type FormEvent, type ReactNode } from 'react'
import { useT } from '../i18n'
import { errorMessage } from '../lib/errors'
import { Button } from './Button'
import styles from './Sheet.module.css'

interface SheetFormProps {
  title: string
  submit: string
  pendingLabel?: string
  canSubmit: boolean
  hint?: ReactNode
  onSubmit: () => Promise<void>
  onClose: () => void
  children: ReactNode
}

export function SheetForm({ title, submit, pendingLabel, canSubmit, hint, onSubmit, onClose, children }: SheetFormProps) {
  const t = useT()
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string>()

  const send = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit || pending) return
    setPending(true)
    setError(undefined)
    try {
      await onSubmit()
      onClose()
    } catch (err) {
      setError(errorMessage(err))
      setPending(false)
    }
  }

  return (
    <Dialog.Root open onOpenChange={(open) => !open && !pending && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.sheet} aria-describedby={undefined}>
          <form onSubmit={(e) => void send(e)}>
            <Dialog.Title className={styles.title}>{title}</Dialog.Title>
            <fieldset className={styles.fields} disabled={pending}>
              {children}
            </fieldset>
            {error ? (
              <p className={styles.error} role="alert">
                {error}
              </p>
            ) : (
              hint && <p className={styles.hint}>{hint}</p>
            )}
            <div className={styles.actions}>
              <Dialog.Close asChild>
                <Button disabled={pending}>{t.common.cancel}</Button>
              </Dialog.Close>
              <Button variant="primary" type="submit" disabled={!canSubmit || pending}>
                {pending && pendingLabel ? pendingLabel : submit}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
