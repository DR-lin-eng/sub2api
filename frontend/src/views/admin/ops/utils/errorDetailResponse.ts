import type { OpsErrorDetail } from '@/api/admin/ops'

const GENERIC_UPSTREAM_MESSAGES = new Set([
  'upstream request failed',
  'upstream request failed after retries',
  'upstream gateway error',
  'upstream service temporarily unavailable'
])

type ParsedGatewayError = {
  type: string
  message: string
}

export type ParsedUpstreamEvent = {
  kind: string
  message: string
  detail: string
  platform: string
  accountId: number | null
  accountName: string
  requestId: string
  statusCode: number | null
  occurredAt: string
  upstreamRequestBody: string
}

function parseGatewayErrorBody(raw: string): ParsedGatewayError | null {
  const text = String(raw || '').trim()
  if (!text) return null

  try {
    const parsed = JSON.parse(text) as Record<string, any>
    const err = parsed?.error as Record<string, any> | undefined
    if (!err || typeof err !== 'object') return null

    const type = typeof err.type === 'string' ? err.type.trim() : ''
    const message = typeof err.message === 'string' ? err.message.trim() : ''
    if (!type && !message) return null

    return { type, message }
  } catch {
    return null
  }
}

function isGenericGatewayUpstreamError(raw: string): boolean {
  const parsed = parseGatewayErrorBody(raw)
  if (!parsed) return false
  if (parsed.type !== 'upstream_error') return false
  return GENERIC_UPSTREAM_MESSAGES.has(parsed.message.toLowerCase())
}

export function resolveUpstreamPayload(
  detail: Pick<OpsErrorDetail, 'upstream_error_detail' | 'upstream_errors' | 'upstream_error_message'> | null | undefined
): string {
  if (!detail) return ''

  const detailPayload = String(detail.upstream_error_detail || '').trim()
  if (detailPayload && detailPayload !== '{}' && detailPayload !== '[]' && detailPayload.toLowerCase() !== 'null') {
    return detailPayload
  }

  const messagePayload = String(detail.upstream_error_message || '').trim()
  if (messagePayload && messagePayload !== '{}' && messagePayload !== '[]' && messagePayload.toLowerCase() !== 'null') {
    return messagePayload
  }

  const upstreamErrorsPayload = String(detail.upstream_errors || '').trim()
  if (!upstreamErrorsPayload || upstreamErrorsPayload === '{}' || upstreamErrorsPayload === '[]' || upstreamErrorsPayload.toLowerCase() === 'null') {
    return ''
  }

  // Do not surface upstream event arrays as the primary response block.
  // They are rendered as split cards in the modal instead.
  const parsedEvents = parseUpstreamEventsPayload(upstreamErrorsPayload)
  if (parsedEvents.length > 0) {
    return ''
  }

  return upstreamErrorsPayload
}

export function parseUpstreamEventsPayload(raw: string): ParsedUpstreamEvent[] {
  const text = String(raw || '').trim()
  if (!text || text === '[]' || text === '{}' || text.toLowerCase() === 'null') {
    return []
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return []
  }

  if (!Array.isArray(parsed)) {
    return []
  }

  return parsed
    .map((item): ParsedUpstreamEvent | null => {
      if (!item || typeof item !== 'object') return null
      const record = item as Record<string, unknown>
      const accountIdRaw = record.account_id
      const statusCodeRaw = record.upstream_status_code
      const atRaw = record.at_unix_ms

      const accountId =
        typeof accountIdRaw === 'number'
          ? accountIdRaw
          : typeof accountIdRaw === 'string' && /^\d+$/.test(accountIdRaw.trim())
            ? Number.parseInt(accountIdRaw.trim(), 10)
            : null

      const statusCode =
        typeof statusCodeRaw === 'number'
          ? statusCodeRaw
          : typeof statusCodeRaw === 'string' && /^\d+$/.test(statusCodeRaw.trim())
            ? Number.parseInt(statusCodeRaw.trim(), 10)
            : null

      const occurredAt =
        typeof atRaw === 'number' && Number.isFinite(atRaw)
          ? new Date(atRaw).toISOString()
          : ''

      return {
        kind: typeof record.kind === 'string' ? record.kind.trim() : '',
        message: typeof record.message === 'string' ? record.message.trim() : '',
        detail: typeof record.detail === 'string' ? record.detail.trim() : '',
        platform: typeof record.platform === 'string' ? record.platform.trim() : '',
        accountId,
        accountName: typeof record.account_name === 'string' ? record.account_name.trim() : '',
        requestId: typeof record.upstream_request_id === 'string' ? record.upstream_request_id.trim() : '',
        statusCode,
        occurredAt,
        upstreamRequestBody: typeof record.upstream_request_body === 'string' ? record.upstream_request_body.trim() : '',
      }
    })
    .filter((item): item is ParsedUpstreamEvent => !!item)
}

export function resolvePrimaryResponseBody(
  detail: OpsErrorDetail | null,
  errorType?: 'request' | 'upstream'
): string {
  if (!detail) return ''

  const upstreamPayload = resolveUpstreamPayload(detail)
  const errorBody = String(detail.error_body || '').trim()

  if (errorType === 'upstream') {
    return upstreamPayload || errorBody
  }

  if (!errorBody) {
    return upstreamPayload
  }

  // For request detail modal, keep client-visible body by default.
  // But if that body is a generic gateway wrapper, show upstream payload first.
  if (upstreamPayload && isGenericGatewayUpstreamError(errorBody)) {
    return upstreamPayload
  }

  return errorBody
}
