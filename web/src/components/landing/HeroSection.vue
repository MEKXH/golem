<template>
  <section class="hero-section">
    <div class="hero-grid">
      <div class="hero-copy-column">
        <div class="hero-toolbar">
          <p class="eyebrow">{{ landing.eyebrow }}</p>
          <LocaleSwitch />
        </div>
        <h1 class="hero-title">{{ landing.title }}</h1>
        <p class="hero-copy">{{ landing.copy }}</p>
        <div class="hero-actions">
          <RouterLink class="button button-primary" to="/console">{{ landing.enterConsole }}</RouterLink>
          <a class="button button-ghost" href="#geo-evolution">{{ landing.seeGeo }}</a>
        </div>
      </div>
      <div class="hero-card-stage">
        <div class="hero-card-stack">
          <article class="signal-card signal-card-primary">
            <span class="signal-label">{{ landing.heroCards[0].label }}</span>
            <strong>{{ landing.heroCards[0].title }}</strong>
            <p>{{ landing.heroCards[0].body }}</p>
          </article>
          <article class="signal-card">
            <span class="signal-label">{{ landing.heroCards[1].label }}</span>
            <strong>{{ landing.heroCards[1].title }}</strong>
            <p>{{ landing.heroCards[1].body }}</p>
          </article>
          <article class="signal-card signal-card-accent">
            <span class="signal-label">{{ landing.heroCards[2].label }}</span>
            <strong>{{ landing.heroCards[2].title }}</strong>
            <p>{{ landing.heroCards[2].body }}</p>
          </article>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LocaleSwitch from '../LocaleSwitch.vue'
import { useLocale } from '../../lib/locale'

const { copy } = useLocale()
const landing = computed(() => copy.value.landing)
</script>
