<template>
  <section class="timeline-panel">
    <article v-for="entry in entries" :key="entry.id" class="chat-entry" :class="`chat-entry-${entry.role}`">
      <div>
        <strong>{{ entry.title }}</strong>
        <p>{{ entry.body }}</p>
      </div>
      <span>{{ entry.meta ?? entry.role }}</span>
    </article>
    <article v-if="isSending" class="chat-entry chat-entry-system">
      <div>
        <strong>{{ consoleCopy.timeline.sendingTitle }}</strong>
        <p>{{ consoleCopy.timeline.sendingBody }}</p>
      </div>
      <span>{{ consoleCopy.timeline.sendingMeta }}</span>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useLocale } from '../../lib/locale'
import type { ChatEntry } from '../../types'

defineProps<{
  entries: ChatEntry[]
  isSending: boolean
}>()

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)
</script>
