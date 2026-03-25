<template>
  <section class="timeline-panel" role="log" aria-live="polite" ref="panelRef">
    <article v-for="entry in entries" :key="entry.id" class="chat-entry" :class="`chat-entry-${entry.role}`">
      <div>
        <strong>{{ entry.title }}</strong>
        <p>{{ entry.body }}</p>
      </div>
      <span>{{ entry.meta ?? entry.role }}</span>
    </article>
    <article v-if="isSending" class="chat-entry chat-entry-system" aria-busy="true">
      <div>
        <strong>{{ consoleCopy.timeline.sendingTitle }}</strong>
        <p>{{ consoleCopy.timeline.sendingBody }}</p>
      </div>
      <span>{{ consoleCopy.timeline.sendingMeta }}</span>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useLocale } from '../../lib/locale'
import type { ChatEntry } from '../../types'

const props = defineProps<{
  entries: ChatEntry[]
  isSending: boolean
}>()

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)

const panelRef = ref<HTMLElement | null>(null)

watch([() => props.entries.length, () => props.isSending], () => {
  nextTick(() => {
    if (panelRef.value) {
      panelRef.value.scrollTop = panelRef.value.scrollHeight
    }
  })
})
</script>
