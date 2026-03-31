import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SettingsView from '../SettingsView.vue'

const {
  settingsAPI,
  groupsGetAll,
  showError,
  showSuccess,
  fetchPublicSettings,
  adminSettingsFetch
} = vi.hoisted(() => ({
  settingsAPI: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    testSmtpConnection: vi.fn(),
    sendTestEmail: vi.fn(),
    getAdminApiKey: vi.fn(),
    regenerateAdminApiKey: vi.fn(),
    deleteAdminApiKey: vi.fn(),
    getOverloadCooldownSettings: vi.fn(),
    updateOverloadCooldownSettings: vi.fn(),
    getStreamTimeoutSettings: vi.fn(),
    updateStreamTimeoutSettings: vi.fn(),
    getRectifierSettings: vi.fn(),
    updateRectifierSettings: vi.fn(),
    getBetaPolicySettings: vi.fn(),
    updateBetaPolicySettings: vi.fn(),
    getTLSFingerprintSettings: vi.fn(),
    updateTLSFingerprintSettings: vi.fn(),
    createTLSFingerprintProfile: vi.fn(),
    updateTLSFingerprintProfile: vi.fn(),
    deleteTLSFingerprintProfile: vi.fn()
  },
  groupsGetAll: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn(),
  adminSettingsFetch: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: settingsAPI,
    groups: {
      getAll: groupsGetAll
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    fetchPublicSettings
  })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: adminSettingsFetch
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('admin SettingsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    settingsAPI.getSettings.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: false,
      registration_email_suffix_whitelist: [],
      promo_code_enabled: true,
      password_reset_enabled: false,
      frontend_url: '',
      invitation_code_enabled: false,
      totp_enabled: false,
      totp_encryption_key_configured: false,
      default_balance: 0,
      default_concurrency: 1,
      default_subscriptions: [],
      site_name: 'Sub2API',
      site_logo: '',
      site_subtitle: '',
      api_base_url: '',
      contact_info: '',
      doc_url: '',
      home_content: '',
      hide_ccs_import_button: false,
      purchase_subscription_enabled: false,
      purchase_subscription_url: '',
      sora_client_enabled: false,
      backend_mode_enabled: false,
      custom_menu_items: [],
      smtp_host: '',
      smtp_port: 587,
      smtp_username: '',
      smtp_password_configured: false,
      smtp_from_email: '',
      smtp_from_name: '',
      smtp_use_tls: true,
      turnstile_enabled: false,
      turnstile_site_key: '',
      turnstile_secret_key_configured: false,
      linuxdo_connect_enabled: false,
      linuxdo_connect_client_id: '',
      linuxdo_connect_client_secret_configured: false,
      linuxdo_connect_redirect_url: '',
      enable_model_fallback: false,
      fallback_model_anthropic: '',
      fallback_model_openai: '',
      fallback_model_gemini: '',
      fallback_model_antigravity: '',
      enable_identity_patch: true,
      identity_patch_prompt: '',
      ops_monitoring_enabled: true,
      ops_realtime_monitoring_enabled: true,
      ops_query_mode_default: 'auto',
      ops_metrics_interval_seconds: 60,
      min_claude_code_version: '',
      max_claude_code_version: '',
      allow_ungrouped_key_scheduling: false,
      auto_delete_401_accounts: false,
      auto_delete_429_accounts: false,
      auto_delete_useless_proxies: false
    })
    settingsAPI.getAdminApiKey.mockResolvedValue({ exists: false, masked_key: '' })
    settingsAPI.getOverloadCooldownSettings.mockResolvedValue({ enabled: true, cooldown_minutes: 10 })
    settingsAPI.getStreamTimeoutSettings.mockResolvedValue({
      enabled: false,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10
    })
    settingsAPI.getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true
    })
    settingsAPI.getBetaPolicySettings.mockResolvedValue({ rules: [] })
    settingsAPI.getTLSFingerprintSettings.mockResolvedValue({
      enabled: true,
      items: [
        {
          profile_id: 'alpha',
          name: 'Alpha',
          enabled: true,
          enable_grease: false,
          cipher_suites: [4866, 4867],
          curves: [29, 23],
          point_formats: [],
          updated_at: '2026-03-28T08:00:00Z'
        }
      ]
    })
    groupsGetAll.mockResolvedValue([])
  })

  it('loads and renders TLS fingerprint profiles in gateway tab', async () => {
    const wrapper = mount(SettingsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
          Select: { template: '<select><slot /></select>' },
          GroupBadge: true,
          GroupOptionItem: true,
          Toggle: { template: '<input type="checkbox" />' },
          ImageUpload: true,
          BackupSettings: true,
          DataManagementSettings: true
        }
      }
    })

    await flushPromises()

    expect(settingsAPI.getTLSFingerprintSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Alpha')
    expect(wrapper.text()).toContain('alpha')
    expect(wrapper.text()).toContain('admin.settings.tlsFingerprint.title')
  })
})
