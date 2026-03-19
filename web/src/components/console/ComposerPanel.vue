<template>
  <form class="composer-panel" @submit.prevent="$emit('submit-prompt')">
    <textarea :value="modelValue" rows="5" placeholder="Send a prompt to the Gateway chat endpoint." @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"></textarea>
    <div class="composer-actions">
      <button class="button button-primary" type="submit" :disabled="isSending">{{ isSending ? 'Sending...' : 'Send Prompt' }}</button>
      <button class="button button-ghost" type="button" :disabled="isChecking" @click="$emit('check-gateway')">{{ isChecking ? 'Checking...' : 'Check Gateway' }}</button>
    </div>
  </form>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string
  isSending: boolean
  isChecking: boolean
}>()

defineEmits<{
  'update:modelValue': [string]
  'submit-prompt': []
  'check-gateway': []
}>()
</script>
