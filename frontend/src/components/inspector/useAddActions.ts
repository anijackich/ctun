import { ClipboardPaste, FileUp, Link, type LucideIcon } from 'lucide-react'
import { useT } from '../../i18n'
import { readText } from '../../lib/clipboard'
import { looksLikeImport } from '../../lib/url'
import { useApp } from '../../store/app'

export interface AddAction {
  icon: LucideIcon
  label: string
  short?: string
  hint?: string
  run: () => void
}

export function useAddActions(openSubscription: () => void): AddAction[] {
  const t = useT()
  const importFile = useApp((s) => s.importFile)
  const importText = useApp((s) => s.importText)
  const setNotice = useApp((s) => s.setNotice)

  const importClipboard = async () => {
    const text = await readText()
    if (looksLikeImport(text)) await importText(text)
    else setNotice(t.inspector.clipboardEmpty)
  }

  return [
    { icon: Link, label: t.inspector.addSubscription, run: openSubscription },
    { icon: FileUp, label: t.inspector.addFile, short: t.empty.fromFile, run: () => void importFile() },
    {
      icon: ClipboardPaste,
      label: t.inspector.addClipboard,
      short: t.empty.fromClipboard,
      hint: '⌘V',
      run: () => void importClipboard(),
    },
  ]
}
