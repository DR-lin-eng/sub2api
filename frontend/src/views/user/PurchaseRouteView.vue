<template>
  <component :is="resolvedView" />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const legacyView = defineAsyncComponent(() => import('./PurchaseSubscriptionView.vue'))
const paymentView = defineAsyncComponent(() => import('./PaymentView.vue'))

const resolvedView = computed(() => (
  appStore.cachedPublicSettings?.payment_enabled ? paymentView : legacyView
))
</script>
