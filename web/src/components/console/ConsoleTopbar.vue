<template>
  <header class="console-topbar">
    <div class="console-topbar-copy">
      <p class="eyebrow">{{ consoleCopy.eyebrow }}</p>
      <h1>{{ consoleCopy.title }}</h1>
      <p class="console-subtitle">{{ consoleCopy.subtitle }}</p>
    </div>
    <div class="console-topbar-actions">
      <span class="status-pill" :class="statusClass">{{ healthStatus }}</span>
      <span v-if="versionState !== null" class="status-pill status-pill-neutral">{{ versionState.version }}</span>
      <LocaleSwitch />
      <button class="button button-ghost button-compact" type="button" :disabled="isChecking" @click="$emit('refresh')">
        {{ isChecking ? consoleCopy.refreshing : consoleCopy.refresh }}
      </button>
      <RouterLink class="button button-ghost button-compact" to="/">{{ consoleCopy.backToLanding }}</RouterLink>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LocaleSwitch from '../LocaleSwitch.vue'
import { useLocale } from '../../lib/locale'
import type { VersionState } from '../../types'

const props = defineProps<{
  healthStatus: string
  versionState: VersionState | null
  isChecking: boolean
}>()

defineEmits<{
  refresh: []
}>()

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)

const statusClass = computed(() => {
  if (props.healthStatus === 'ok') {
    return 'status-pill-live'
  }
  if (props.healthStatus === 'unknown') {
    return 'status-pill-neutral'
  }
  return 'status-pill-error'
})
</script>
