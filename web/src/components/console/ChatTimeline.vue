<template>
  <section class="timeline-panel" ref="scrollContainer" role="log" aria-live="polite" :aria-busy="isSending">
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
import { computed, ref, watch } from 'vue'
import { useLocale } from '../../lib/locale'
import type { ChatEntry } from '../../types'

const props = defineProps<{
  entries: ChatEntry[]
  isSending: boolean
}>()

const scrollContainer = ref<HTMLElement | null>(null)

watch(
  [() => props.entries.length, () => props.isSending],
  () => {
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  },
  { flush: 'post' }
)

const { copy } = useLocale()
const consoleCopy = computed(() => copy.value.console)
</script>
