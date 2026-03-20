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
import { useLocale } from '../lib/locale'
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
const { copy, locale } = useLocale()
const consoleCopy = computed(() => copy.value.console)
const entries = ref<ChatEntry[]>([])
const draft = ref('Analyze the current workspace and tell me what changed in the Geo stack.')
const isChecking = ref(false)
const isSending = ref(false)

const healthStatus = computed(() => healthState.value?.status ?? 'unknown')

function buildIntroEntry(): ChatEntry {
  return {
    id: 'system-intro',
    role: 'system',
    title: consoleCopy.value.timeline.introTitle,
    body: consoleCopy.value.timeline.introBody,
    meta: consoleCopy.value.timeline.introMeta,
  }
}

function ensureIntroEntry() {
  if (entries.value.length === 0) {
    entries.value = [buildIntroEntry()]
    return
  }
  const firstEntry = entries.value[0]
  if (firstEntry?.id === 'system-intro') {
    entries.value = [buildIntroEntry(), ...entries.value.slice(1)]
  }
}

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
      title: consoleCopy.value.timeline.gatewayCheckPassedTitle,
      body: `Health is ${health.status}. Connected version is ${version.version}.`,
      meta: `health ${health.request_id}`,
    })
  } catch (error) {
    appendEntry({
      id: randomId('error'),
      role: 'error',
      title: consoleCopy.value.timeline.gatewayCheckFailedTitle,
      body: error instanceof Error ? error.message : 'Unknown connection error',
      meta: consoleCopy.value.timeline.checkMeta,
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
    title: consoleCopy.value.timeline.promptTitle,
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
      title: consoleCopy.value.timeline.responseTitle,
      body: response.response,
      meta: response.request_id,
    })
  } catch (error) {
    appendEntry({
      id: randomId('error'),
      role: 'error',
      title: consoleCopy.value.timeline.requestFailedTitle,
      body: error instanceof Error ? error.message : 'Unknown request failure',
      meta: consoleCopy.value.timeline.chatMeta,
    })
  } finally {
    isSending.value = false
  }
}

watch(settings, saveSettings, { deep: true })
watch(locale, ensureIntroEntry)

onMounted(() => {
  const stored = localStorage.getItem(storageKey)
  if (stored !== null) {
    try {
      settings.value = { ...settings.value, ...(JSON.parse(stored) as Partial<GatewaySettings>) }
    } catch {
      localStorage.removeItem(storageKey)
    }
  }
  ensureIntroEntry()
})
</script>
