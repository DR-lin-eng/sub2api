export interface ParsedOpenAIRefreshTokenInput {
  refreshToken: string
  clientId?: string
}

function sanitizeValue(raw: string): string {
  return raw.trim().replace(/^["'`]+|["'`,;]+$/g, '')
}

function buildParsedInput(refreshToken?: string, clientId?: string): ParsedOpenAIRefreshTokenInput | null {
  const rt = sanitizeValue(refreshToken || '')
  if (!rt) {
    return null
  }
  const parsed: ParsedOpenAIRefreshTokenInput = { refreshToken: rt }
  const cid = sanitizeValue(clientId || '')
  if (cid) {
    parsed.clientId = cid
  }
  return parsed
}

function tryParseJSONLike(raw: string): ParsedOpenAIRefreshTokenInput | null {
  const trimmed = raw.trim()
  if (!trimmed) return null

  const candidates = [trimmed]
  const firstBrace = trimmed.indexOf('{')
  const lastBrace = trimmed.lastIndexOf('}')
  if (firstBrace >= 0 && lastBrace > firstBrace) {
    candidates.push(trimmed.slice(firstBrace, lastBrace + 1))
  }

  for (const candidate of candidates) {
    try {
      const parsed = JSON.parse(candidate) as Record<string, unknown>
      if (!parsed || typeof parsed !== 'object') continue
      const refreshToken =
        (typeof parsed.refresh_token === 'string' && parsed.refresh_token) ||
        (typeof parsed.rt === 'string' && parsed.rt) ||
        ''
      const clientId =
        (typeof parsed.client_id === 'string' && parsed.client_id) ||
        ''
      const result = buildParsedInput(refreshToken, clientId)
      if (result) return result
    } catch {
      // ignore
    }
  }
  return null
}

function tryParseQueryLike(raw: string): ParsedOpenAIRefreshTokenInput | null {
  const trimmed = raw.trim()
  if (!trimmed) return null

  const candidates = [trimmed]
  const queryIndex = trimmed.indexOf('?')
  if (queryIndex >= 0 && queryIndex < trimmed.length-1) {
    candidates.push(trimmed.slice(queryIndex + 1))
  }
  const hashIndex = trimmed.indexOf('#')
  if (hashIndex >= 0 && hashIndex < trimmed.length-1) {
    candidates.push(trimmed.slice(hashIndex + 1))
  }

  for (const candidate of candidates) {
    const params = new URLSearchParams(candidate)
    const refreshToken = params.get('refresh_token') || params.get('rt') || ''
    const clientId = params.get('client_id') || ''
    const result = buildParsedInput(refreshToken, clientId)
    if (result) return result
  }
  return null
}

function tryParseRegexLike(raw: string): ParsedOpenAIRefreshTokenInput | null {
  const refreshMatch = raw.match(/(?:^|[\s,{])(?:refresh_token|rt)\s*[:=]\s*["']?([^"',\s&#}]+)/i)
  const clientMatch = raw.match(/(?:^|[\s,{])client_id\s*[:=]\s*["']?([^"',\s&#}]+)/i)
  return buildParsedInput(refreshMatch?.[1], clientMatch?.[1])
}

export function parseOpenAIRefreshTokenInput(raw: string): ParsedOpenAIRefreshTokenInput | null {
  const trimmed = raw.trim()
  if (!trimmed) return null

  return (
    tryParseJSONLike(trimmed) ||
    tryParseQueryLike(trimmed) ||
    tryParseRegexLike(trimmed) ||
    buildParsedInput(trimmed)
  )
}

export function parseOpenAIRefreshTokenInputs(raw: string): ParsedOpenAIRefreshTokenInput[] {
  const trimmed = raw.trim()
  if (!trimmed) return []

  const wholeParsed = tryParseJSONLike(trimmed) || tryParseQueryLike(trimmed)
  if (wholeParsed) {
    return [wholeParsed]
  }

  return trimmed
    .split('\n')
    .map((line) => parseOpenAIRefreshTokenInput(line))
    .filter((item): item is ParsedOpenAIRefreshTokenInput => item !== null)
}
