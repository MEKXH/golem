export interface GatewayErrorPayload {
  code?: string
  message?: string
  request_id?: string
}

export interface GatewayHealthResponse {
  status: string
  request_id: string
}

export interface GatewayVersionResponse {
  version: string
  request_id: string
}

export interface GatewayChatResponse {
  response: string
  session_id: string
  request_id: string
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function buildHeaders(token: string): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  }
  const trimmed = token.trim()
  if (trimmed !== '') {
    headers.Authorization = `Bearer ${trimmed}`
  }
  return headers
}

async function parseJson<T>(response: Response): Promise<T> {
  return (await response.json()) as T
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (response.ok) {
    return parseJson<T>(response)
  }
  let payload: GatewayErrorPayload = {}
  try {
    payload = await parseJson<GatewayErrorPayload>(response)
  } catch {
    payload = {}
  }
  throw new Error(payload.message ?? `Request failed with status ${response.status}`)
}

export async function fetchHealth(baseUrl: string, token: string): Promise<GatewayHealthResponse> {
  const response = await fetch(`${trimTrailingSlash(baseUrl)}/health`, {
    method: 'GET',
    headers: buildHeaders(token),
  })
  return handleResponse<GatewayHealthResponse>(response)
}

export async function fetchVersion(baseUrl: string, token: string): Promise<GatewayVersionResponse> {
  const response = await fetch(`${trimTrailingSlash(baseUrl)}/version`, {
    method: 'GET',
    headers: buildHeaders(token),
  })
  return handleResponse<GatewayVersionResponse>(response)
}

export async function sendChat(
  baseUrl: string,
  token: string,
  sessionId: string,
  senderId: string,
  message: string,
): Promise<GatewayChatResponse> {
  const response = await fetch(`${trimTrailingSlash(baseUrl)}/chat`, {
    method: 'POST',
    headers: buildHeaders(token),
    body: JSON.stringify({
      message,
      session_id: sessionId,
      sender_id: senderId,
    }),
  })
  return handleResponse<GatewayChatResponse>(response)
}
