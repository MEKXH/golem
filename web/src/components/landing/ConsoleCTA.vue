<template>
  <section class="cta-section">
    <div class="cta-card">
      <p class="eyebrow">{{ landing.cta.eyebrow }}</p>
      <h2>{{ landing.cta.title }}</h2>
      <p>{{ landing.cta.copy }}</p>
      <RouterLink class="button button-primary" to="/console">{{ landing.cta.action }}</RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../lib/locale'

const { copy } = useLocale()
const landing = computed(() => copy.value.landing)
</script>
