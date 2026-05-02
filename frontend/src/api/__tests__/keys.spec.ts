import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.fn()

vi.mock('../client', () => ({
  apiClient: {
    post: postMock
  }
}))

describe('keys API create', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('includes allowed_models in create payload', async () => {
    postMock.mockResolvedValue({ data: { id: 1 } })
    const mod = await import('../keys')

    await mod.create(
      'test-key',
      1,
      [1],
      ['gpt-5.4', 'claude-sonnet-4-5'],
      'sk-custom',
      ['127.0.0.1'],
      [],
      10,
      30,
      { rate_limit_1d: 5 }
    )

    expect(postMock).toHaveBeenCalledWith('/keys', expect.objectContaining({
      name: 'test-key',
      group_id: 1,
      group_ids: [1],
      allowed_models: ['gpt-5.4', 'claude-sonnet-4-5'],
      custom_key: 'sk-custom'
    }))
  })
})
