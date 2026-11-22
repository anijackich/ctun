import { Check, Copy } from 'lucide-react'
import { useEffect, useState } from 'react'
import { copyText } from '../lib/clipboard'
import { Button } from './Button'

const COPIED_MS = 1500

interface CopyButtonProps {
  text: () => Promise<string>
  label: string
}

export function CopyButton({ text, label }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), COPIED_MS)
    return () => clearTimeout(timer)
  }, [copied])

  const copy = () =>
    text()
      .then(copyText)
      .then(() => setCopied(true), () => {})

  return (
    <Button variant="plain" icon onClick={() => void copy()} aria-label={label} title={label}>
      {copied ? <Check size={13} /> : <Copy size={13} />}
    </Button>
  )
}
