<template>
  <div class="flex flex-wrap items-center gap-3">
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.accounts.searchAccounts')"
      class="w-full sm:w-64"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />
    <Select
      :model-value="filters.platform || ''"
      class="w-40"
      :options="platformOptions"
      @update:model-value="updatePlatform"
      @change="$emit('change')"
    />
    <Select
      :model-value="filters.type || ''"
      class="w-40"
      :options="typeOptions"
      @update:model-value="updateField('type', $event)"
      @change="$emit('change')"
    />
    <Select
      :model-value="filters.status || ''"
      class="w-40"
      :options="statusOptions"
      @update:model-value="updateField('status', $event)"
      @change="$emit('change')"
    />
    <Select
      :model-value="filters.group || ''"
      class="w-40"
      :options="groupOptions"
      @update:model-value="updateField('group', $event)"
      @change="$emit('change')"
    />
    <Select
      v-if="showPlanFilter"
      :model-value="filters.plan || ''"
      class="w-36"
      :options="planOptions"
      @update:model-value="updateField('plan', $event)"
      @change="$emit('change')"
    />
    <Select
      v-if="showOAuthTypeFilter"
      :model-value="filters.oauth_type || ''"
      class="w-40"
      :options="oauthTypeOptions"
      @update:model-value="updateField('oauth_type', $event)"
      @change="$emit('change')"
    />
    <input
      v-if="showTierFilter"
      :value="filters.tier_id || ''"
      class="input w-36"
      :placeholder="t('admin.accounts.tierIdPlaceholder')"
      @input="updateTierID"
      @change="$emit('change')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import type { AdminGroup, AccountListFilters } from '@/types'

type AccountFiltersModel = AccountListFilters & { group?: string }

const props = defineProps<{
  searchQuery: string
  filters: AccountFiltersModel
  groups?: AdminGroup[]
}>()

const emit = defineEmits<{
  'update:searchQuery': [value: string]
  'update:filters': [value: AccountFiltersModel]
  change: []
}>()

const { t } = useI18n()

const showPlanFilter = computed(() => !props.filters.platform || props.filters.platform === 'openai')
const showOAuthTypeFilter = computed(() => !props.filters.platform || props.filters.platform === 'gemini')
const showTierFilter = computed(() => !props.filters.platform || props.filters.platform === 'gemini')

const updateField = (field: keyof AccountFiltersModel, value: string | number | boolean | null) => {
  emit('update:filters', {
    ...props.filters,
    [field]: value == null ? '' : String(value)
  })
}

const updatePlatform = (value: string | number | boolean | null) => {
  const nextPlatform = value == null ? '' : String(value)
  const nextFilters: AccountFiltersModel = {
    ...props.filters,
    platform: nextPlatform
  }
  if (nextPlatform && nextPlatform !== 'openai') {
    nextFilters.plan = ''
  }
  if (nextPlatform && nextPlatform !== 'gemini') {
    nextFilters.oauth_type = ''
    nextFilters.tier_id = ''
  }
  emit('update:filters', nextFilters)
}

const updateTierID = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('update:filters', {
    ...props.filters,
    tier_id: target.value
  })
}

const platformOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'kiro', label: 'Kiro' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'sora', label: 'Sora' }
])

const typeOptions = computed(() => [
  { value: '', label: t('admin.accounts.allTypes') },
  { value: 'oauth', label: t('admin.accounts.oauthType') },
  { value: 'setup-token', label: t('admin.accounts.setupToken') },
  { value: 'apikey', label: t('admin.accounts.apiKey') },
  { value: 'bedrock', label: 'AWS Bedrock' }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.accounts.allStatus') },
  { value: 'active', label: t('admin.accounts.status.active') },
  { value: 'inactive', label: t('admin.accounts.status.inactive') },
  { value: 'error', label: t('admin.accounts.status.error') },
  { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') },
  { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }
])

const groupOptions = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...((props.groups || []).map((group) => ({ value: String(group.id), label: group.name })))
])

const planOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlans') },
  { value: 'free', label: t('admin.accounts.planFree') },
  { value: 'plus', label: t('admin.accounts.planPlus') },
  { value: 'team', label: t('admin.accounts.planTeam') }
])

const oauthTypeOptions = computed(() => [
  { value: '', label: t('admin.accounts.allOAuthTypes') },
  { value: 'code_assist', label: 'Code Assist' },
  { value: 'google_one', label: 'Google One' },
  { value: 'ai_studio', label: 'AI Studio' }
])
</script>
