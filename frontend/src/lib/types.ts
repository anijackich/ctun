const PAGES = ['dashboard', 'routing', 'connections', 'logs', 'settings'] as const
export type Page = (typeof PAGES)[number]

export type CoreStatus = 'stopped' | 'starting' | 'running' | 'stopping'

export const TUNNEL_MODES = ['tun', 'proxy'] as const
export type TunnelMode = (typeof TUNNEL_MODES)[number]

export const ROUTING_MODES = ['rule', 'global', 'direct'] as const
export type RoutingMode = (typeof ROUTING_MODES)[number]

export const LOG_LEVELS = ['error', 'warning', 'info', 'debug'] as const
export type LogLevel = (typeof LOG_LEVELS)[number]

export const TUN_STACKS = ['system', 'gvisor', 'mixed'] as const
export type TunStack = (typeof TUN_STACKS)[number]

export const THEMES = ['', 'light', 'dark'] as const
export type Theme = (typeof THEMES)[number]

export const TRAY_ICONS = ['', 'flag', 'logo'] as const
export type TrayIcon = (typeof TRAY_ICONS)[number]

export type HelperState = 'installed' | 'missing' | 'outdated' | 'unsupported'

export interface DnsSettings {
  nameservers: string[]
  direct: string[]
  servers: string[]
  viaProxy: boolean
  systemOverride: boolean
}

export const DNS_LISTS = ['nameservers', 'direct', 'servers'] as const
export type DnsList = (typeof DNS_LISTS)[number]

export interface AppSettings {
  locale: string
  theme: Theme
  trayIcon: TrayIcon
  tunnelMode: TunnelMode
  routingMode: RoutingMode
  mixedPort: number
  portAuth: boolean
  systemProxy: boolean
  logLevel: LogLevel
  tunStack: TunStack
  inspectorOpen: boolean
  inspectorWidth: number
  dns: DnsSettings
}

export type SettingsPatch = Partial<AppSettings>

export type CoreErrorCode =
  | 'tun-unavailable'
  | 'core-missing'
  | 'core-failed'
  | 'no-profile'
  | 'profile-invalid'
  | 'helper-missing'
  | 'helper-outdated'
  | 'system-proxy'

export interface CoreState {
  status: CoreStatus
  startedAt?: number
  error?: string
  errorCode?: CoreErrorCode
}

export interface TrafficSample {
  up: number
  down: number
  upTotal: number
  downTotal: number
}

export interface Traffic {
  up: number
  down: number
}

export interface Session {
  uploaded: number
  downloaded: number
  connections: number
  memory: number
}

export interface ExitInfo {
  ip: string
  country: string
  city: string
  org: string
}

export interface SubscriptionUsage {
  upload: number
  download: number
  total: number
  expire?: number
}

export interface Profile {
  id: string
  name: string
  host: string
  updatedAt: number
  active: boolean
  local: boolean
  userAgent: string
  updateIntervalHours: number
  usage: SubscriptionUsage
  supportUrl?: string
  webPageUrl?: string
  announce?: string
}

export interface NewProfile {
  url: string
  name: string
  userAgent: string
  updateIntervalHours: number
}

export interface ProxyNode {
  name: string
  protocol: string
  delay?: number
}

export interface ProxyGroup {
  name: string
  type: 'select' | 'url-test' | 'fallback' | 'load-balance'
  now: string
  nodes: ProxyNode[]
}

export interface LatencySample {
  profileId: string
  name: string
  delay: number
}

export type LogSource = 'core' | 'app'

export interface LogLine {
  time: number
  source: LogSource
  level: LogLevel
  message: string
}

export interface Connection {
  id: string
  host: string
  port: string
  network: string
  inbound: string
  process: string
  processPath: string
  pid: number
  chain: string[]
  rule: string
  rulePayload: string
  upload: number
  download: number
  start: number
}

export interface Rule {
  index: number
  type: string
  payload: string
  target: string
  hits: number
  lastHit: number
  disabled: boolean
}

export interface RuleSet {
  name: string
  behavior: string
  format: string
  source: string
  rules: number
  updatedAt: number
}

export interface Routing {
  live: boolean
  rules: Rule[]
  ruleSets: RuleSet[]
}

export type RouteTarget = 'block' | 'direct' | 'proxy'

export interface RouteList {
  target: RouteTarget
  entries: string[]
}

export interface RoutingProfile {
  id: string
  name: string
  lists: RouteList[]
  proxyLan: boolean
  final: RouteTarget
}

export interface EntryError {
  target: RouteTarget
  line: number
  entry: string
  message: string
}

export interface RoutingProfileState {
  profile: RoutingProfile
  active: boolean
  errors: EntryError[]
}
