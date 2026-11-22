import { useEffect, useState, type CSSProperties } from 'react'
import { Inspector } from './components/inspector/Inspector'
import { NavRail } from './components/NavRail'
import { Notice } from './components/Notice'
import { ResizeHandle } from './components/ResizeHandle'
import { useT } from './i18n'
import { looksLikeImport } from './lib/url'
import { useAccent } from './lib/useAccent'
import { Connections } from './pages/Connections'
import { Dashboard } from './pages/dashboard/Dashboard'
import { Logs } from './pages/Logs'
import { Routing } from './pages/routing/Routing'
import { Settings } from './pages/Settings'
import { useApp } from './store/app'
import styles from './App.module.css'

const isEditable = (target: EventTarget | null) =>
  target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement

export function App() {
  const page = useApp((s) => s.page)
  const t = useT()
  const inspectorOpen = useApp((s) => s.inspectorOpen)
  const inspectorWidth = useApp((s) => s.inspectorWidth)
  const setInspectorWidth = useApp((s) => s.setInspectorWidth)
  const [resizing, setResizing] = useState(false)
  const importText = useApp((s) => s.importText)
  useAccent()

  useEffect(() => {
    const cleanup = useApp.getState().init()
    return () => void cleanup.then((off) => off())
  }, [])

  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      if (isEditable(e.target)) return
      const text = e.clipboardData?.getData('text') ?? ''
      if (looksLikeImport(text)) void importText(text)
    }
    document.addEventListener('paste', onPaste)
    return () => document.removeEventListener('paste', onPaste)
  }, [importText])

  return (
    <div className={styles.window}>
      <NavRail />
      <main className={styles.main} key={page}>
        {page === 'dashboard' ? (
          <Dashboard />
        ) : page === 'connections' ? (
          <Connections />
        ) : page === 'logs' ? (
          <Logs />
        ) : page === 'routing' ? (
          <Routing />
        ) : (
          <Settings />
        )}
      </main>
      <div
        className={styles.inspectorSlot}
        data-open={inspectorOpen || undefined}
        data-resizing={resizing || undefined}
        style={{ '--inspector-width': `${inspectorWidth}px` } as CSSProperties}
      >
        {inspectorOpen && (
          <ResizeHandle
            size={inspectorWidth}
            onResize={setInspectorWidth}
            onResizingChange={setResizing}
            label={t.inspector.resize}
          />
        )}
        <Inspector />
      </div>
      <Notice />
    </div>
  )
}
