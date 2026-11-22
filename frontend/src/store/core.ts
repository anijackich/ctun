import type { StateCreator } from 'zustand'
import { backend, privilege } from '../lib/backend'
import { errorMessage } from '../lib/errors'
import type { CoreErrorCode, CoreState, CoreStatus, ExitInfo, HelperState, Session, Traffic } from '../lib/types'
import type { AppState } from './app'

export const TRAFFIC_HISTORY = 60
const LIVE_GROUPS_REFRESH_MS = 15_000
// Let the core settle on the new route before asking for the exit address.
const EXIT_LOOKUP_DELAY_MS = 1200

const EMPTY_SESSION: Session = { uploaded: 0, downloaded: 0, connections: 0, memory: 0 }

export interface CoreError {
  code?: CoreErrorCode
  message: string
}

export interface CoreSlice {
  status: CoreStatus
  startedAt?: number
  error?: CoreError
  traffic: Traffic[]
  session: Session
  exit?: ExitInfo
  helper?: HelperState
  helperBusy: boolean

  toggleConnection: () => void
  installHelper: () => Promise<void>
  uninstallHelper: () => Promise<void>
  refreshExit: () => void
  attachCore: () => Promise<() => void>
}

let liveGroupsTimer: ReturnType<typeof setInterval> | undefined
let exitTimer: ReturnType<typeof setTimeout> | undefined

export const createCoreSlice: StateCreator<AppState, [], [], CoreSlice> = (set, get) => {
  const refreshHelper = async () => set({ helper: await privilege.status() })

  const refreshActiveGroups = () => {
    const id = get().activeProfileId
    if (id) void get().loadGroups(id)
  }

  // resync restarts the timers and lookups even without a status change: a re-attach (React
  // StrictMode, hot reload) has cleared them.
  const applyCoreState = (state: CoreState, resync = false) => {
    const running = state.status === 'running'
    if (state.errorCode === 'helper-missing') set({ helper: 'missing' })
    if (state.errorCode === 'helper-outdated') set({ helper: 'outdated' })
    if (running && get().settings?.tunnelMode === 'tun') set({ helper: 'installed' })
    const changed = get().status !== state.status
    set((s) => ({
      status: state.status,
      startedAt: running ? state.startedAt : undefined,
      error: state.error ? { code: state.errorCode, message: state.error } : undefined,
      traffic: running || state.status === 'stopping' ? s.traffic : [],
      session: state.status === 'starting' ? EMPTY_SESSION : s.session,
    }))
    if (!changed && !resync) return
    clearInterval(liveGroupsTimer)
    if (running) {
      liveGroupsTimer = setInterval(refreshActiveGroups, LIVE_GROUPS_REFRESH_MS)
      get().refreshExit()
    } else {
      set({ exit: undefined })
    }
    if (running || state.status === 'stopped') refreshActiveGroups()
  }

  return {
    status: 'stopped',
    traffic: [],
    session: EMPTY_SESSION,
    helperBusy: false,

    toggleConnection: () => {
      const { status, settings, helper, helperBusy } = get()
      if (status === 'running') {
        void backend.stop().catch(() => {})
        return
      }
      if (status !== 'stopped' || helperBusy) return
      set({ error: undefined })
      if (settings?.tunnelMode === 'tun' && (helper === 'missing' || helper === 'outdated')) {
        void get().installHelper()
        return
      }
      void backend.start().catch(() => {})
    },

    installHelper: async () => {
      set({ helperBusy: true })
      let installed = false
      try {
        await privilege.install()
        installed = true
        set({ error: undefined })
      } catch (err) {
        set({ error: { code: 'helper-missing', message: errorMessage(err) } })
      } finally {
        set({ helperBusy: false })
      }
      if (!installed) return
      await refreshHelper()
      const { status, settings } = get()
      if (status === 'stopped' && settings?.tunnelMode === 'tun') get().toggleConnection()
    },

    uninstallHelper: async () => {
      set({ helperBusy: true })
      try {
        await privilege.uninstall()
      } catch (err) {
        set({ error: { message: errorMessage(err) } })
      } finally {
        set({ helperBusy: false })
      }
      await refreshHelper()
    },

    refreshExit: () => {
      clearTimeout(exitTimer)
      set({ exit: undefined })
      exitTimer = setTimeout(() => {
        if (get().status !== 'running') return
        backend
          .exitInfo()
          .then((exit) => get().status === 'running' && set({ exit }))
          .catch(() => {})
      }, EXIT_LOOKUP_DELAY_MS)
    },

    attachCore: async () => {
      const offs = [
        backend.onStatus((state) => applyCoreState(state)),
        backend.onTraffic((t) =>
          set((s) => ({
            traffic: [...s.traffic.slice(1 - TRAFFIC_HISTORY), { up: t.up, down: t.down }],
            session: { ...s.session, uploaded: t.upTotal, downloaded: t.downTotal },
          })),
        ),
        backend.onStats((stats) => set((s) => ({ session: { ...s.session, ...stats } }))),
      ]
      applyCoreState(await backend.coreState(), true)
      void refreshHelper()
      return () => {
        clearInterval(liveGroupsTimer)
        clearTimeout(exitTimer)
        offs.forEach((off) => off())
      }
    },
  }
}
