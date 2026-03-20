<template>
  <aside class="console-side-card">
    <div class="side-card-header">
      <p class="eyebrow">{{ consoleCopy.connection.eyebrow }}</p>
      <h2>{{ consoleCopy.connection.title }}</h2>
    </div>
    <div class="field-stack">
      <label>
        <span>{{ consoleCopy.connection.baseUrl }}</span>
        <input :value="modelValue.baseUrl" @input="patch('baseUrl', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>{{ consoleCopy.connection.bearerToken }}</span>
        <input :value="modelValue.bearerToken" type="password" :placeholder="consoleCopy.connection.bearerPlaceholder" @input="patch('bearerToken', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>{{ consoleCopy.connection.sessionId }}</span>
        <input :value="modelValue.sessionId" @input="patch('sessionId', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>{{ consoleCopy.connection.senderId }}</span>
        <input :value="modelValue.senderId" @input="patch('senderId', ($event.target as HTMLInputElement).value)" />
      </label>
    </div>
    <div class="console-note">{{ consoleCopy.connection.note }}</div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../lib/locale'
import type { GatewaySettings } from '../../types'

const props = defineProps<{
  modelValue: GatewaySettings
}>()

const emit = defineEmits<{
  'update:modelValue': [GatewaySettings]
}>()

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)

function patch<Key extends keyof GatewaySettings>(key: Key, value: GatewaySettings[Key]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  })
}
</script>
