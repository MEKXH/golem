<template>
  <header class="console-topbar">
    <div>
      <p class="eyebrow">Console</p>
      <h1>Gateway Control Surface</h1>
      <p class="console-subtitle">Connect to health, version, and chat from one operator-grade workspace.</p>
    </div>
    <div class="topbar-pills">
      <span class="status-pill" :class="statusClass">{{ healthStatus }}</span>
      <span v-if="versionState !== null" class="status-pill status-pill-neutral">{{ versionState.version }}</span>
      <button class="button button-ghost button-compact" type="button" :disabled="isChecking" @click="$emit('refresh')">
        {{ isChecking ? 'Checking...' : 'Refresh Gateway' }}
      </button>
      <RouterLink class="button button-ghost button-compact" to="/">Back to Landing</RouterLink>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { VersionState } from '../../types'

const props = defineProps<{
  healthStatus: string
  versionState: VersionState | null
  isChecking: boolean
}>()

defineEmits<{
  refresh: []
}>()

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
