import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.fn()
const put = vi.fn()
const post = vi.fn()
const del = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    put,
    post,
    delete: del
  }
}))

describe('admin settings api tls fingerprint', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('requests tls fingerprint settings and profiles from dedicated endpoints', async () => {
    get.mockResolvedValue({ data: { enabled: true, items: [] } })
    put.mockResolvedValue({ data: { enabled: true } })
    post.mockResolvedValue({ data: { profile_id: 'alpha' } })

    const mod = await import('../settings')

    await mod.getTLSFingerprintSettings()
    await mod.listTLSFingerprintProfiles()
    await mod.updateTLSFingerprintSettings({ enabled: true })
    await mod.createTLSFingerprintProfile({
      profile_id: 'alpha',
      name: 'Alpha',
      enabled: true,
      enable_grease: false,
      cipher_suites: [],
      curves: [],
      point_formats: []
    })

    expect(get).toHaveBeenNthCalledWith(1, '/admin/settings/tls-fingerprint')
    expect(get).toHaveBeenNthCalledWith(2, '/admin/settings/tls-fingerprint/profiles')
    expect(put).toHaveBeenCalledWith('/admin/settings/tls-fingerprint', { enabled: true })
    expect(post).toHaveBeenCalledWith('/admin/settings/tls-fingerprint/profiles', expect.objectContaining({
      profile_id: 'alpha'
    }))
  })
})
