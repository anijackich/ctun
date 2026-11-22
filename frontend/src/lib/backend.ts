import { Events } from '@wailsio/runtime'
import {
  AppService,
  CoreService,
  PrivilegeService,
  ProfileService,
  RoutingService,
  SettingsService,
} from '../../bindings/ctun/internal/app'
import type {
  GroupView,
  ProfileView,
  RoutingProfileView,
  SettingsPatch as WirePatch,
} from '../../bindings/ctun/internal/app/models'
import {
  CoreLogs,
  CoreStats,
  CoreStatus,
  CoreTraffic,
  LatencyResult,
  ProfilesChanged,
  RoutingChanged,
  SettingsChanged,
} from '../../bindings/events'
import type {
  AppSettings,
  Connection,
  CoreState,
  ExitInfo,
  HelperState,
  LatencySample,
  LogLine,
  NewProfile,
  Profile,
  ProxyGroup,
  Routing,
  RoutingProfile,
  RoutingProfileState,
  RouteTarget,
  SettingsPatch,
  TrafficSample,
} from './types'

/** The only module touching generated bindings; Go enums are narrowed to literal types here. */

interface CoreRuntimeStats {
  connections: number
  memory: number
}

const toSettings = (s: unknown) => s as AppSettings
const toState = (s: unknown) => s as CoreState

export const backend = {
  appVersion: () => AppService.Version(),
  settings: () => SettingsService.Get().then(toSettings),
  updateSettings: (patch: SettingsPatch) =>
    SettingsService.Update(patch as unknown as WirePatch).then(toSettings),
  resetDNS: () => SettingsService.ResetDNS().then(toSettings),
  invalidNameservers: (lines: string[]) => SettingsService.InvalidNameservers(lines).then((l) => l ?? []),

  coreState: () => CoreService.State().then(toState),
  start: () => CoreService.Start(),
  stop: () => CoreService.Stop(),
  proxyURL: () => CoreService.ProxyURL(),
  systemProxySupported: () => CoreService.SystemProxySupported(),
  coreVersion: () => CoreService.CoreVersion(),
  exitInfo: (): Promise<ExitInfo> => CoreService.ExitInfo(),

  onStatus: (cb: (state: CoreState) => void) => Events.On(CoreStatus, (e) => cb(toState(e.data))),
  onTraffic: (cb: (sample: TrafficSample) => void) => Events.On(CoreTraffic, (e) => cb(e.data)),
  onStats: (cb: (stats: CoreRuntimeStats) => void) => Events.On(CoreStats, (e) => cb(e.data)),

  logs: () => CoreService.Logs().then((lines) => (lines ?? []) as LogLine[]),
  clearLogs: () => CoreService.ClearLogs(),
  onLogs: (cb: (lines: LogLine[]) => void) => Events.On(CoreLogs, (e) => cb(e.data as LogLine[])),

  connections: () =>
    CoreService.Connections().then((list) =>
      (list ?? []).map((c): Connection => ({ ...c, chain: c.chain ?? [] })),
    ),
  closeConnection: (id: string) => CoreService.CloseConnection(id),
  closeAllConnections: () => CoreService.CloseAllConnections(),
  onSettings: (cb: (settings: AppSettings) => void) =>
    Events.On(SettingsChanged, (e) => cb(toSettings(e.data))),
}

const toProfile = (v: ProfileView): Profile => ({
  id: v.id,
  name: v.name,
  host: v.host,
  updatedAt: v.updatedAt,
  active: v.active,
  local: v.local,
  userAgent: v.userAgent,
  updateIntervalHours: v.updateIntervalHours,
  usage: {
    upload: v.info.upload,
    download: v.info.download,
    total: v.info.total,
    expire: v.info.expire ? v.info.expire * 1000 : undefined,
  },
  supportUrl: v.info.supportUrl || undefined,
  webPageUrl: v.info.webPageUrl || undefined,
  announce: v.info.announce || undefined,
})

const toGroups = (groups: GroupView[] | null): ProxyGroup[] =>
  (groups ?? []).map((g) => ({
    name: g.name,
    type: g.type as ProxyGroup['type'],
    now: g.now,
    nodes: (g.nodes ?? []).map((n) => ({ name: n.name, protocol: n.protocol, delay: n.delay ?? undefined })),
  }))

const toProfiles = (list: ProfileView[] | null) => (list ?? []).map(toProfile)

export const profiles = {
  list: () => ProfileService.List().then(toProfiles),
  add: (p: NewProfile) => ProfileService.Add(p).then(toProfile),
  edit: (p: NewProfile & { id: string }) => ProfileService.Edit(p).then(toProfile),
  import: (text: string) => ProfileService.Import(text).then(toProfile),
  importFile: () => ProfileService.ImportFile().then((v) => (v ? toProfile(v) : undefined)),
  refresh: (id: string) => ProfileService.Refresh(id),
  remove: (id: string) => ProfileService.Remove(id),
  link: (id: string) => ProfileService.SubscriptionURL(id),
  groups: (id: string) => ProfileService.Groups(id).then(toGroups),
  select: (id: string, group: string, name: string) => ProfileService.Select(id, group, name),
  testLatency: (id: string, group: string) => ProfileService.TestLatency(id, group).then(toGroups),
  onChange: (cb: (list: Profile[]) => void) =>
    Events.On(ProfilesChanged, (e) => cb(toProfiles(e.data))),
  onLatency: (cb: (result: LatencySample) => void) => Events.On(LatencyResult, (e) => cb(e.data)),
}

export const routing = {
  get: () =>
    RoutingService.Routing().then(
      (v): Routing => ({ live: v.live, rules: v.rules ?? [], ruleSets: v.ruleSets ?? [] }),
    ),
  setRuleDisabled: (index: number, disabled: boolean) => RoutingService.SetRuleDisabled(index, disabled),
  updateRuleSet: (name: string) => RoutingService.UpdateRuleSet(name),

  profiles: () => RoutingService.Profiles().then(toRoutingProfiles),
  saveProfile: (p: RoutingProfile) =>
    RoutingService.SaveProfile(p as unknown as RoutingProfileView['profile']).then(toRoutingProfile),
  createProfile: (name: string, from = '') => RoutingService.CreateProfile(name, from).then(toRoutingProfile),
  deleteProfile: (id: string) => RoutingService.DeleteProfile(id),
  activateProfile: (id: string) => RoutingService.ActivateProfile(id),
  onProfiles: (cb: (profiles: RoutingProfileState[]) => void) =>
    Events.On(RoutingChanged, (e) => cb(toRoutingProfiles(e.data))),
}

function toRoutingProfile(v: RoutingProfileView): RoutingProfileState {
  const p = v.profile
  return {
    active: v.active,
    errors: (v.errors ?? []) as RoutingProfileState['errors'],
    profile: {
      id: p.id,
      name: p.name,
      proxyLan: p.proxyLan,
      final: p.final as RouteTarget,
      lists: (p.lists ?? []).map((l) => ({
        target: l.target as RouteTarget,
        entries: l.entries ?? [],
      })),
    },
  }
}

const toRoutingProfiles = (list: RoutingProfileView[] | null) => (list ?? []).map(toRoutingProfile)

export const privilege = {
  status: () => PrivilegeService.Status().then((s) => s.state as HelperState),
  install: () => PrivilegeService.Install(),
  uninstall: () => PrivilegeService.Uninstall(),
}
