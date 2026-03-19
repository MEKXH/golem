<template>
  <aside class="console-side-card">
    <div class="side-card-header">
      <p class="eyebrow">Connection</p>
      <h2>Gateway Settings</h2>
    </div>
    <div class="field-stack">
      <label>
        <span>Base URL</span>
        <input :value="modelValue.baseUrl" @input="patch('baseUrl', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>Bearer Token</span>
        <input :value="modelValue.bearerToken" type="password" placeholder="optional" @input="patch('bearerToken', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>Session ID</span>
        <input :value="modelValue.sessionId" @input="patch('sessionId', ($event.target as HTMLInputElement).value)" />
      </label>
      <label>
        <span>Sender ID</span>
        <input :value="modelValue.senderId" @input="patch('senderId', ($event.target as HTMLInputElement).value)" />
      </label>
    </div>
    <div class="console-note">
      These values stay local in the browser. The first release supports bearer-token based Gateway access without adding a separate auth system.
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { GatewaySettings } from '../../types'

const props = defineProps<{
  modelValue: GatewaySettings
}>()

const emit = defineEmits<{
  'update:modelValue': [GatewaySettings]
}>()

function patch<Key extends keyof GatewaySettings>(key: Key, value: GatewaySettings[Key]) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  })
}
</script>
