import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsRequestDetailsModal from '../OpsRequestDetailsModal.vue'

const mockListRequestDetails = vi.fn()
const mockShowError = vi.fn()
const mockCopyToClipboard = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestDetails: (...args: any[]) => mockListRequestDetails(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockShowError,
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: mockCopyToClipboard,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, any>) => {
        if (key === 'admin.ops.requestDetails.table.ttft') return 'TTFT'
        if (key === 'admin.ops.requestDetails.table.duration') return 'Duration'
        if (key === 'admin.ops.requestDetails.table.time') return 'Time'
        if (key === 'admin.ops.requestDetails.table.kind') return 'Kind'
        if (key === 'admin.ops.requestDetails.table.platform') return 'Platform'
        if (key === 'admin.ops.requestDetails.table.model') return 'Model'
        if (key === 'admin.ops.requestDetails.table.status') return 'Status'
        if (key === 'admin.ops.requestDetails.table.requestId') return 'Request ID'
        if (key === 'admin.ops.requestDetails.table.actions') return 'Actions'
        if (key === 'admin.ops.requestDetails.kind.success') return 'SUCCESS'
        if (key === 'admin.ops.requestDetails.kind.error') return 'ERROR'
        if (key === 'common.refresh') return 'Refresh'
        if (key === 'common.loading') return 'Loading'
        if (key === 'admin.ops.requestDetails.empty') return 'Empty'
        if (key === 'admin.ops.requestDetails.emptyHint') return 'Empty hint'
        if (key === 'admin.ops.requestDetails.copy') return 'Copy'
        if (key === 'admin.ops.requestDetails.rangeMinutes' && params) return `${params.n} minutes`
        if (key === 'admin.ops.requestDetails.rangeHours' && params) return `${params.n} hours`
        if (key === 'admin.ops.requestDetails.rangeLabel' && params) return `Window: ${params.range}`
        return key
      },
    }),
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialogStub',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  template: `
    <div v-if="show" class="dialog-stub">
      <div class="dialog-title">{{ title }}</div>
      <slot />
    </div>
  `,
})

const PaginationStub = defineComponent({
  name: 'PaginationStub',
  template: '<div class="pagination-stub" />',
})

const baseResponse = {
  items: [
    {
      kind: 'success' as const,
      created_at: '2026-04-26T10:00:00Z',
      request_id: 'req_ttft_1',
      platform: 'openai',
      model: 'gpt-4o-mini',
      duration_ms: 987,
      first_token_ms: 123,
      status_code: 200,
      stream: true,
    },
  ],
  total: 1,
  page: 1,
  page_size: 10,
}

function mountModal(sort: 'first_token_desc' | 'duration_desc') {
  return mount(OpsRequestDetailsModal, {
    props: {
      modelValue: false,
      timeRange: '30m',
      preset: {
        title: sort === 'first_token_desc' ? 'TTFT Detail' : 'Duration Detail',
        sort,
      },
      platform: 'openai',
      groupId: 7,
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Pagination: PaginationStub,
      },
    },
  })
}

describe('OpsRequestDetailsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListRequestDetails.mockResolvedValue(baseResponse)
    mockCopyToClipboard.mockResolvedValue(true)
  })

  it('TTFT 视图请求 first_token_desc 并显示 first_token_ms', async () => {
    const wrapper = mountModal('first_token_desc')

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(mockListRequestDetails).toHaveBeenCalledWith(
      expect.objectContaining({
        sort: 'first_token_desc',
        platform: 'openai',
        group_id: 7,
        page: 1,
        page_size: 10,
      })
    )

    expect(wrapper.text()).toContain('TTFT')
    expect(wrapper.text()).toContain('123 ms')
    expect(wrapper.text()).not.toContain('987 ms')
  })

  it('耗时视图继续显示 duration_ms', async () => {
    const wrapper = mountModal('duration_desc')

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(mockListRequestDetails).toHaveBeenCalledWith(
      expect.objectContaining({
        sort: 'duration_desc',
      })
    )

    expect(wrapper.text()).toContain('Duration')
    expect(wrapper.text()).toContain('987 ms')
    expect(wrapper.text()).not.toContain('123 ms')
  })
})
