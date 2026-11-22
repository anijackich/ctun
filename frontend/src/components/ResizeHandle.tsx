import type { KeyboardEvent, PointerEvent as ReactPointerEvent } from 'react'
import styles from './ResizeHandle.module.css'

interface ResizeHandleProps {
  size: number
  onResize: (size: number) => void
  onResizingChange?: (resizing: boolean) => void
  label: string
}

const KEY_STEP = 16

export function ResizeHandle({ size, onResize, onResizingChange, label }: ResizeHandleProps) {
  const onPointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return
    e.preventDefault()
    const startX = e.clientX
    const startSize = size

    const move = (ev: PointerEvent) => onResize(startSize + startX - ev.clientX)
    const stop = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', stop)
      window.removeEventListener('pointercancel', stop)
      onResizingChange?.(false)
    }

    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', stop)
    window.addEventListener('pointercancel', stop)
    onResizingChange?.(true)
  }

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'ArrowLeft') onResize(size + KEY_STEP)
    else if (e.key === 'ArrowRight') onResize(size - KEY_STEP)
  }

  return (
    <div
      className={styles.handle}
      role="separator"
      tabIndex={0}
      aria-orientation="vertical"
      aria-valuenow={size}
      aria-label={label}
      onPointerDown={onPointerDown}
      onKeyDown={onKeyDown}
    />
  )
}
