import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshOpenAIToken: vi.fn(),
      validateSoraSessionToken: vi.fn(),
      inspectOpenAIAccessToken: vi.fn()
    }
  }
}))

import { useOpenAIOAuth } from '@/composables/useOpenAIOAuth'
import { adminAPI } from '@/api/admin'

const refreshOpenAITokenMock = vi.mocked(adminAPI.accounts.refreshOpenAIToken)
const inspectOpenAIAccessTokenMock = vi.mocked(adminAPI.accounts.inspectOpenAIAccessToken)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useOpenAIOAuth.buildCredentials', () => {
  it('should keep client_id when token response contains it', () => {
    const oauth = useOpenAIOAuth({ platform: 'sora' })
    const creds = oauth.buildCredentials({
      access_token: 'at',
      refresh_token: 'rt',
      client_id: 'app_sora_client',
      expires_at: 1700000000
    })

    expect(creds.client_id).toBe('app_sora_client')
    expect(creds.access_token).toBe('at')
    expect(creds.refresh_token).toBe('rt')
  })

  it('should keep legacy behavior when client_id is missing', () => {
    const oauth = useOpenAIOAuth({ platform: 'openai' })
    const creds = oauth.buildCredentials({
      access_token: 'at',
      refresh_token: 'rt',
      expires_at: 1700000000
    })

    expect(Object.prototype.hasOwnProperty.call(creds, 'client_id')).toBe(false)
    expect(creds.access_token).toBe('at')
    expect(creds.refresh_token).toBe('rt')
  })
})

describe('useOpenAIOAuth.validateRefreshToken', () => {
  it('should pass parsed client_id when refresh token input is JSON', async () => {
    refreshOpenAITokenMock.mockResolvedValueOnce({ access_token: 'at', refresh_token: 'rt' })
    const oauth = useOpenAIOAuth({ platform: 'openai' })

    await oauth.validateRefreshToken(
      JSON.stringify({
        refresh_token: 'rt-json',
        client_id: 'app_custom_client'
      })
    )

    expect(refreshOpenAITokenMock).toHaveBeenCalledWith(
      'rt-json',
      undefined,
      '/admin/openai/refresh-token',
      'app_custom_client'
    )
  })

  it('should keep supporting plain refresh tokens', async () => {
    refreshOpenAITokenMock.mockResolvedValueOnce({ access_token: 'at', refresh_token: 'rt' })
    const oauth = useOpenAIOAuth({ platform: 'openai' })

    await oauth.validateRefreshToken('rt-plain')

    expect(refreshOpenAITokenMock).toHaveBeenCalledWith(
      'rt-plain',
      undefined,
      '/admin/openai/refresh-token',
      undefined
    )
  })

  it('should prefer error.message from api client interceptor errors', async () => {
    refreshOpenAITokenMock.mockRejectedValueOnce({ message: 'token refresh failed: status 502' })
    const oauth = useOpenAIOAuth({ platform: 'openai' })

    const result = await oauth.validateRefreshToken('rt-plain')

    expect(result).toBeNull()
    expect(oauth.error.value).toBe('token refresh failed: status 502')
  })
})

describe('useOpenAIOAuth.validateAccessToken', () => {
  it('should validate access token through at2info endpoint', async () => {
    inspectOpenAIAccessTokenMock.mockResolvedValueOnce({ access_token: 'at', client_id: 'app_chatweb' })
    const oauth = useOpenAIOAuth({ platform: 'openai' })

    await oauth.validateAccessToken('at-token')

    expect(inspectOpenAIAccessTokenMock).toHaveBeenCalledWith(
      'at-token',
      '/admin/openai/at2info'
    )
  })

  it('should prefer interceptor error.message for access token validation', async () => {
    inspectOpenAIAccessTokenMock.mockRejectedValueOnce({ message: 'access token inspect failed: status 400' })
    const oauth = useOpenAIOAuth({ platform: 'openai' })

    const result = await oauth.validateAccessToken('at-token')

    expect(result).toBeNull()
    expect(oauth.error.value).toBe('access token inspect failed: status 400')
  })
})
