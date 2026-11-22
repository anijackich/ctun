import { create } from 'zustand'
import { createCoreSlice, type CoreSlice } from './core'
import { createProfilesSlice, type ProfilesSlice } from './profiles'
import { createUiSlice, type UiSlice } from './ui'

export type AppState = UiSlice &
  CoreSlice &
  ProfilesSlice & {
    init: () => Promise<() => void>
  }

export const useApp = create<AppState>()((...a) => ({
  ...createUiSlice(...a),
  ...createCoreSlice(...a),
  ...createProfilesSlice(...a),

  // Settings first: the core state is read against the tunnel mode.
  init: async () => {
    const get = a[1]
    const offs = [await get().attachUi(), await get().attachProfiles(), await get().attachCore()]
    return () => offs.forEach((off) => off())
  },
}))
