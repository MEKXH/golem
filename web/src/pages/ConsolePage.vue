<template>
  <main class="console-page">
    <ConsoleTopbar
      :health-status="healthStatus"
      :version-state="versionState"
      :is-checking="isChecking"
      @refresh="checkGateway"
    />
    <section class="console-layout">
      <ConnectionPanel v-model="settings" />
      <div class="console-main-card">
        <ChatTimeline :entries="entries" :is-sending="isSending" />
        <ComposerPanel
          v-model="draft"
          :is-sending="isSending"
          :is-checking="isChecking"
          @submit-prompt="submitPrompt"
          @check-gateway="checkGateway"
        />
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import ChatTimeline from '../components/console/ChatTimeline.vue'
import ComposerPanel from '../components/console/ComposerPanel.vue'
import ConnectionPanel from '../components/console/ConnectionPanel.vue'
import ConsoleTopbar from '../components/console/ConsoleTopbar.vue'
import { fetchHealth, fetchVersion, sendChat } from '../lib/api'
import type { ChatEntry, GatewaySettings, HealthState, VersionState } from '../types'

const storageKey = 'golem-web-ui-settings'

const settings = ref<GatewaySettings>({
  baseUrl: 'http://127.0.0.1:18790',
  bearerToken: '',
  sessionId: 'default',
  senderId: 'web-ui',
})
const versionState = ref<VersionState | null>(null)
const healthState = ref<HealthState | null>(null)
const entries = ref<ChatEntry[]>([
  {
    id: 'system-intro',
    role: 'system',
    title: 'Console Ready',
    body: 'Check Gateway connectivity, inspect version, then send a prompt through the existing /chat API.',
    meta: 'idle',
  },
])
const draft = ref('Analyze the current workspace and tell me what changed in the Geo stack.')
const isChecking = ref(false)
const isSending = ref(false)

const healthStatus = computed(() => healthState.value?.status ?? 'unknown')

function appendEntry(entry: ChatEntry) {
  entries.value = [...entries.value, entry]
}

function randomId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
}

function saveSettings() {
  localStorage.setItem(storageKey, JSON.stringify(settings.value))
}

async function checkGateway() {
  isChecking.value = true
  try {
    const [health, version] = await Promise.all([
      fetchHealth(settings.value.baseUrl, settings.value.bearerToken),
      fetchVersion(settings.value.baseUrl, settings.value.bearerToken),
    ])
    healthState.value = { status: health.status, requestId: health.request_id }
    versionState.value = { version: version.version, requestId: version.request_id }
    appendEntry({
      id: randomId('system'),
      role: 'system',
      title: 'Gateway Check Passed',
      body: `Health is ${health.status}. Connected version is ${version.version}.`,
      meta: `health ${health.request_id}`,
    })
  } catch (error) {
    appendEntry({
      id: randomId('error'),
      role: 'error',
      title: 'Gateway Check Failed',
      body: error instanceof Error ? error.message : 'Unknown connection error',
      meta: 'check failed',
    })
  } finally {
    isChecking.value = false
  }
}

async function submitPrompt() {
  const message = draft.value.trim()
  if (message === '' || isSending.value) {
    return
  }

  appendEntry({
    id: randomId('user'),
    role: 'user',
    title: 'Prompt',
    body: message,
    meta: settings.value.sessionId,
  })

  isSending.value = true
  draft.value = ''
  try {
    const response = await sendChat(
      settings.value.baseUrl,
      settings.value.bearerToken,
      settings.value.sessionId,
      settings.value.senderId,
      message,
    )
    appendEntry({
      id: randomId('assistant'),
      role: 'assistant',
      title: 'Response',
      body: response.response,
      meta: response.request_id,
    })
  } catch (error) {
    appendEntry({
      id: randomId('error'),
      role: 'error',
      title: 'Request Failed',
      body: error instanceof Error ? error.message : 'Unknown request failure',
      meta: 'chat error',
    })
  } finally {
    isSending.value = false
  }
}

watch(settings, saveSettings, { deep: true })

onMounted(() => {
  const stored = localStorage.getItem(storageKey)
  if (stored !== null) {
    try {
      settings.value = { ...settings.value, ...(JSON.parse(stored) as Partial<GatewaySettings>) }
    } catch {
      localStorage.removeItem(storageKey)
    }
  }
})
</script>
