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
        <strong>Sending</strong>
        <p>The Gateway request is in flight.</p>
      </div>
      <span>pending</span>
    </article>
  </section>
</template>

<script setup lang="ts">
import type { ChatEntry } from '../../types'

defineProps<{
  entries: ChatEntry[]
  isSending: boolean
}>()
</script>
