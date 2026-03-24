<template>
  <div class="locale-switch" role="group" :aria-label="copy.localeLabel">
    <button
      class="locale-switch-button"
      :class="{ 'locale-switch-button-active': activeLocale === 'en' }"
      type="button"
      :aria-pressed="activeLocale === 'en'"
      @click="setLocale('en')"
    >
      {{ copy.localeEnglish }}
    </button>
    <button
      class="locale-switch-button"
      :class="{ 'locale-switch-button-active': activeLocale === 'zh-CN' }"
      type="button"
      :aria-pressed="activeLocale === 'zh-CN'"
      @click="setLocale('zh-CN')"
    >
      {{ copy.localeChinese }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../lib/locale'

const { locale, copy, setLocale } = useLocale()
const activeLocale = computed(() => locale.value)
</script>
