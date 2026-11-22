import type { StateCreator } from 'zustand'
import { detectLocale, LOCALES, type Locale } from '../i18n/locale'
import { backend } from '../lib/backend'
import { errorMessage } from '../lib/errors'
import type { AppSettings, Page, RoutingMode, SettingsPatch, TunnelMode } from '../lib/types'
import type { AppState } from './app'

const INSPECTOR_WIDTH = { min: 280, default: 320, max: 520 } as const
const WIDTH_SAVE_DELAY_MS = 400

export interface UiSlice {
  page: Page
  locale: Locale
  settings?: AppSettings
  inspectorOpen: boolean
  inspectorWidth: number
  notice?: string

  navigate: (page: Page) => void
  updateSettings: (patch: SettingsPatch) => void
  setTunnelMode: (mode: TunnelMode) => void
  setRoutingMode: (mode: RoutingMode) => void
  setLocale: (locale: Locale) => void
  toggleInspector: () => void
  setInspectorWidth: (width: number) => void
  setNotice: (notice?: string) => void
  attachUi: () => Promise<() => void>
}

const resolveLocale = (locale: string): Locale =>
  locale in LOCALES ? (locale as Locale) : detectLocale()

let widthSaveTimer: ReturnType<typeof setTimeout> | undefined

export const createUiSlice: StateCreator<AppState, [], [], UiSlice> = (set, get) => {
  // Saved settings echo back on every change; the panel layout is local state, so an echo of a
  // debounced width save must not snap a resize in progress.
  const applySettings = (settings: AppSettings) => set({ settings, locale: resolveLocale(settings.locale) })
  const persist = (patch: SettingsPatch) =>
    backend.updateSettings(patch).then(applySettings, (err: unknown) => set({ notice: errorMessage(err) }))

  return {
    page: 'dashboard',
    locale: detectLocale(),
    inspectorOpen: true,
    inspectorWidth: INSPECTOR_WIDTH.default,

    navigate: (page) => set({ page }),

    updateSettings: (patch) => {
      const current = get().settings
      if (current) set({ settings: { ...current, ...patch } })
      void persist(patch)
    },

    setTunnelMode: (tunnelMode) => get().updateSettings({ tunnelMode }),
    setRoutingMode: (routingMode) => get().updateSettings({ routingMode }),
    setLocale: (locale) => {
      set({ locale })
      get().updateSettings({ locale })
    },

    toggleInspector: () => {
      const inspectorOpen = !get().inspectorOpen
      set({ inspectorOpen })
      void persist({ inspectorOpen })
    },

    setInspectorWidth: (width) => {
      const inspectorWidth = Math.round(Math.min(INSPECTOR_WIDTH.max, Math.max(INSPECTOR_WIDTH.min, width)))
      set({ inspectorWidth })
      clearTimeout(widthSaveTimer)
      widthSaveTimer = setTimeout(() => void persist({ inspectorWidth }), WIDTH_SAVE_DELAY_MS)
    },

    setNotice: (notice) => set({ notice }),

    attachUi: async () => {
      const off = backend.onSettings(applySettings)
      const settings = await backend.settings()
      applySettings(settings)
      set({ inspectorOpen: settings.inspectorOpen, inspectorWidth: settings.inspectorWidth })
      return off
    },
  }
}
