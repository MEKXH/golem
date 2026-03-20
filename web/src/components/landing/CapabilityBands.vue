<template>
  <section class="capability-section">
    <div class="section-heading">
      <p class="eyebrow">{{ landing.capability.eyebrow }}</p>
      <h2>{{ landing.capability.title }}</h2>
    </div>
    <div class="capability-grid">
      <article v-for="(item, index) in landing.capability.items" :key="item.title" class="capability-card">
        <span class="capability-index">{{ `0${index + 1}` }}</span>
        <h3>{{ item.title }}</h3>
        <p>{{ item.body }}</p>
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
