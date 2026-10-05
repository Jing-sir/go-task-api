<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { formatDateTime } from '@/utils/datetime'
import type { Task } from '@/api/types'

defineProps<{
  tasks: Task[]
  loading?: boolean
  total?: number
}>()

const { t } = useI18n()
</script>

<template>
  <section class="task-feed">
    <header class="task-feed__header">
      <div>
        <div class="task-feed__eyebrow">{{ t('tasks.total') }}</div>
        <h2 class="task-feed__title">{{ total ?? 0 }}</h2>
      </div>
      <p class="task-feed__hint">{{ t('tasks.subtitle') }}</p>
    </header>

    <div v-if="loading" class="task-feed__state">
      <a-spin />
      <span>{{ t('tasks.loading') }}</span>
    </div>

    <a-empty v-else-if="tasks.length === 0" :description="t('tasks.empty')" />

    <ul v-else class="task-feed__list">
      <li v-for="task in tasks" :key="task.id" class="task-feed__item">
        <div class="task-feed__main">
          <div class="task-feed__title-row">
            <strong class="task-feed__task-title">{{ task.title }}</strong>
            <a-tag :color="task.status ? 'green' : 'gray'">
              {{ task.status ? t('tasks.status.done') : t('tasks.status.todo') }}
            </a-tag>
          </div>

          <p class="task-feed__desc">
            {{ task.description || '—' }}
          </p>
        </div>

        <div class="task-feed__meta">
          <span>{{ formatDateTime(task.created_at) }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.task-feed {
  display: grid;
  gap: 18px;
  padding: 20px;
  border: 1px solid var(--app-line);
  border-radius: var(--app-radius);
  background: var(--app-surface);
}

.task-feed__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: start;
}

.task-feed__eyebrow {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--app-muted);
}

.task-feed__title {
  margin: 4px 0 0;
  font-size: 28px;
}

.task-feed__hint {
  margin: 0;
  max-width: 40ch;
  color: var(--app-muted);
  line-height: 1.6;
}

.task-feed__state {
  display: grid;
  justify-items: center;
  gap: 12px;
  padding: 40px 20px;
  color: var(--app-muted);
}

.task-feed__list {
  display: grid;
  margin: 0;
  padding: 0;
  list-style: none;
  border-top: 1px solid var(--app-line);
}

.task-feed__item {
  display: grid;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 1px solid var(--app-line);
}

.task-feed__title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.task-feed__task-title {
  min-width: 0;
  font-size: 15px;
}

.task-feed__desc {
  margin: 8px 0 0;
  color: var(--app-muted);
  line-height: 1.7;
  white-space: pre-wrap;
}

.task-feed__meta {
  font-size: 12px;
  color: var(--app-muted);
}

@media (max-width: 640px) {
  .task-feed__header {
    flex-direction: column;
  }

  .task-feed__title-row {
    flex-direction: column;
    align-items: start;
  }
}
</style>

