import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'
import OpsErrorLogTable from '../OpsErrorLogTable.vue'

const mockGetRequestErrorDetail = vi.fn()
const mockGetUpstreamErrorDetail = vi.fn()
const mockListRequestErrorUpstreamErrors = vi.fn()
const mockShowError = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail: (...args: any[]) => mockGetRequestErrorDetail(...args),
    getUpstreamErrorDetail: (...args: any[]) => mockGetUpstreamErrorDetail(...args),
    listRequestErrorUpstreamErrors: (...args: any[]) => mockListRequestErrorUpstreamErrors(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: (...args: any[]) => mockShowError(...args),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>',
})

const IconStub = defineComponent({
  name: 'Icon',
  template: '<span class="icon-stub" />',
})

const TooltipStub = defineComponent({
  name: 'ElTooltip',
  template: '<div><slot /></div>',
})

function makeErrorLog(overrides: Record<string, any> = {}) {
  return {
    id: 1,
    created_at: '2026-03-29T12:00:00Z',
    phase: 'upstream',
    type: 'upstream_error',
    error_owner: 'provider',
    error_source: 'upstream_http',
    severity: 'P1',
    status_code: 503,
    platform: 'openai',
    model: 'gpt-5',
    inbound_endpoint: '/v1/chat/completions',
    upstream_endpoint: '/v1/responses',
    requested_model: 'gpt-5',
    upstream_model: 'gpt-5.4-mini',
    request_type: 2,
    is_retryable: true,
    retry_count: 0,
    resolved: false,
    client_request_id: 'creq-1',
    request_id: 'req-1',
    message: 'provider overloaded',
    user_email: 'user@example.com',
    account_name: 'acc-1',
    group_name: 'group-1',
    request_path: '/v1/chat/completions',
    stream: true,
    ...overrides,
  }
}

describe('Ops error observability UI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders endpoint, model mapping, and request type in the error log table', () => {
    const wrapper = mount(OpsErrorLogTable, {
      props: {
        rows: [makeErrorLog()],
        total: 1,
        loading: false,
        page: 1,
        pageSize: 10,
      },
      global: {
        stubs: {
          Pagination: true,
          'el-tooltip': TooltipStub,
        },
      },
    })

    expect(wrapper.text()).toContain('/v1/chat/completions')
    expect(wrapper.text()).toContain('gpt-5')
    expect(wrapper.text()).toContain('gpt-5.4-mini')
    expect(wrapper.text()).toContain('admin.ops.errorLog.requestTypeStream')
  })

  it('renders observability fields in the detail modal and falls back for legacy rows', async () => {
    mockGetRequestErrorDetail.mockResolvedValue(makeErrorLog({
      error_body: '{"error":"bad gateway"}',
      user_agent: 'codex',
      request_body: '{"model":"gpt-5"}',
      request_body_truncated: false,
      is_business_limited: false,
      upstream_error_message: 'provider overloaded',
      upstream_error_detail: '{"provider":"overloaded"}',
      upstream_errors: '',
    }))
    mockListRequestErrorUpstreamErrors.mockResolvedValue({ items: [] })

    const wrapper = mount(OpsErrorDetailModal, {
      props: {
        show: true,
        errorId: 1,
        errorType: 'request',
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: IconStub,
          'el-tooltip': TooltipStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('/v1/chat/completions')
    expect(wrapper.text()).toContain('/v1/responses')
    expect(wrapper.text()).toContain('gpt-5.4-mini')
    expect(wrapper.text()).toContain('admin.ops.errorDetail.requestTypeStream')

    mockGetRequestErrorDetail.mockResolvedValueOnce(makeErrorLog({
      id: 2,
      inbound_endpoint: '',
      upstream_endpoint: '',
      requested_model: '',
      upstream_model: '',
      request_type: null,
      error_body: '{"error":"legacy"}',
      user_agent: 'codex',
      request_body: '',
      request_body_truncated: false,
      is_business_limited: false,
      upstream_error_message: '',
      upstream_error_detail: '',
      upstream_errors: '',
    }))

    await wrapper.setProps({ errorId: 2 })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.ops.errorDetail.requestTypeUnknown')
    expect(wrapper.text()).toContain('gpt-5')
  })
})
