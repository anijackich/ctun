import { Clipboard } from '@wailsio/runtime'

export const copyText = (text: string) => Clipboard.SetText(text)

export const readText = () => Clipboard.Text().catch(() => '')
