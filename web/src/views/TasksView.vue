<script setup lang="ts">
import { computed, onMounted, useTemplateRef } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useI18n } from 'vue-i18n'
import AppShell from '@/components/layout/AppShell.vue'
import TaskComposer from '@/components/tasks/TaskComposer.vue'
import TaskFeed from '@/components/tasks/TaskFeed.vue'
import { getWsUrl } from '@/api/http'
import type { CreateTaskPayload, Task, TaskEvent } from '@/api/types'
import { useTasksStore } from '@/stores/tasks'
import { useWebSocket } from '@/composables/useWebSocket'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const tasksStore = useTasksStore()
const composerRef = useTemplateRef<{ reset: () => void }>('composerRef')

const authStore = useAuthStore()

const socket = useWebSocket({
  url: getWsUrl(authStore.token),
  onMessage(event) {
      const task = JSON.parse(event.data) as TaskEvent

      switch(task.type) {
        case 'task.created':
        case 'task.updated':
          tasksStore.upsert(task.data as Task)
          break
        case 'task.deleted':
          tasksStore.remove(task.data as number)
          break
      }
  },
})

const socketLabel = computed(() => {
  switch (socket.status.value) {
    case 'open':
      return t('tasks.live')
    case 'connecting':
      return t('tasks.idle')
    case 'reconnecting':
      return t('tasks.reconnecting')
    default:
      return t('tasks.offline')
  }
})

const socketColor = computed(() => (socket.status.value === 'open' ? 'green' : 'orange'))

async function refresh() {
  try {
    await tasksStore.loadTasks()
  } catch (error) {
    Message.error(error instanceof Error ? error.message : t('feedback.loadFailed'))
  }
}

async function handleSubmit(payload: CreateTaskPayload) {
  try {
    await tasksStore.createTask(payload)
    await tasksStore.loadTasks()
    composerRef.value?.reset()
    Message.success(t('feedback.created'))
  } catch (error) {
    Message.error(error instanceof Error ? error.message : t('feedback.loadFailed'))
  }
}

onMounted(async () => {
  await refresh()
  socket.open()
})
</script>

<template>
  <AppShell>
    <section class="tasks-page">
      <header class="tasks-page__header">
        <div>
          <div class="tasks-page__eyebrow">{{ t('nav.tasks') }}</div>
          <h1 class="tasks-page__title">{{ t('tasks.title') }}</h1>
          <p class="tasks-page__subtitle">{{ t('tasks.subtitle') }}</p>
        </div>

        <div class="tasks-page__actions">
          <a-tag :color="socketColor">
            {{ socketLabel }}
          </a-tag>
          <a-button @click="refresh">
            {{ t('tasks.refresh') }}
          </a-button>
        </div>
      </header>

      <TaskComposer ref="composerRef" :loading="tasksStore.submitting" @submit="handleSubmit" />

      <TaskFeed
        :tasks="tasksStore.tasks"
        :loading="tasksStore.loading"
        :total="tasksStore.total"
      />
    </section>
  </AppShell>
</template>

<style scoped>
.tasks-page {
  display: grid;
  gap: 20px;
}

.tasks-page__header {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: start;
  padding: 4px 0;
}

.tasks-page__eyebrow {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--app-muted);
}

.tasks-page__title {
  margin: 6px 0 0;
  font-size: 28px;
}

.tasks-page__subtitle {
  margin: 10px 0 0;
  max-width: 52ch;
  color: var(--app-muted);
  line-height: 1.7;
}

.tasks-page__actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

@media (max-width: 760px) {
  .tasks-page__header {
    flex-direction: column;
  }

  .tasks-page__actions {
    flex-wrap: wrap;
  }
}
</style>
