const PLAY_MS = 700

export function installPressFx(root: Document = document) {
  root.addEventListener('pointerdown', (e) => {
    const el = (e.target as Element | null)?.closest<HTMLElement>('[data-fx-on="press"]')
    if (!el) return
    el.removeAttribute('data-playing')
    void el.offsetWidth
    el.setAttribute('data-playing', '')
    window.setTimeout(() => el.removeAttribute('data-playing'), PLAY_MS)
  })
}
