export type ChatRole = 'user' | 'assistant' | 'system' | 'error'

export interface GatewaySettings {
  baseUrl: string
  bearerToken: string
  sessionId: string
  senderId: string
}

export interface VersionState {
  version: string
  requestId: string
}

export interface HealthState {
  status: string
  requestId: string
}

export interface ChatEntry {
  id: string
  role: ChatRole
  title: string
  body: string
  meta?: string
}
