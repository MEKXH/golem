<template>
  <section id="geo-evolution" class="geo-section">
    <div class="section-heading section-heading-wide">
      <p class="eyebrow">{{ landing.geo.eyebrow }}</p>
      <h2>{{ landing.geo.title }}</h2>
      <p>{{ landing.geo.copy }}</p>
    </div>
    <div class="geo-grid">
      <article v-for="panel in landing.geo.panels" :key="panel.title" class="geo-panel">
        <h3>{{ panel.title }}</h3>
        <p>{{ panel.body }}</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../lib/locale'

const { copy } = useLocale()
const landing = computed(() => copy.value.landing)
</script>
