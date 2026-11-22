export const isSubscriptionUrl = (text: string) => /^https?:\/\/\S+$/i.test(text.trim())

export function looksLikeImport(text: string): boolean {
  const t = text.trim()
  return (
    isSubscriptionUrl(t) ||
    /^[a-z][a-z0-9+.-]*:\/\/\S+/im.test(t) ||
    t.startsWith('{') ||
    t.startsWith('[') ||
    /^proxies:/m.test(t)
  )
}
