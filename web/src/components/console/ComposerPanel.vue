<template>
  <form class="composer-panel" @submit.prevent="$emit('submit-prompt')">
    <textarea :value="modelValue" rows="3" :placeholder="consoleCopy.composer.placeholder" :aria-label="consoleCopy.composer.placeholder" @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" @keydown.enter.exact.prevent="$emit('submit-prompt')"></textarea>
    <div class="composer-actions">
      <button class="button button-primary" type="submit" :disabled="isSending">{{ isSending ? consoleCopy.composer.sending : consoleCopy.composer.send }}</button>
      <button class="button button-ghost" type="button" :disabled="isChecking" @click="$emit('check-gateway')">{{ isChecking ? consoleCopy.composer.checking : consoleCopy.composer.check }}</button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../lib/locale'

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

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)
</script>
