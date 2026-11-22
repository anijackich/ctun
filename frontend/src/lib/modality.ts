const NAVIGATION_KEYS = new Set([
  'Tab',
  'ArrowUp',
  'ArrowDown',
  'ArrowLeft',
  'ArrowRight',
  'Home',
  'End',
  'PageUp',
  'PageDown',
])

export function installInputModality() {
  const root = document.documentElement
  root.dataset.input = 'pointer'
  document.addEventListener('pointerdown', () => (root.dataset.input = 'pointer'), true)
  document.addEventListener(
    'keydown',
    (e) => {
      if (NAVIGATION_KEYS.has(e.key)) root.dataset.input = 'keyboard'
    },
    true,
  )
}
