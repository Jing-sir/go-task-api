<script setup lang="ts">
import { reactive } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useI18n } from 'vue-i18n'
import type { CreateTaskPayload } from '@/api/types'

const emit = defineEmits<{
  submit: [CreateTaskPayload]
}>()

defineProps<{
  loading?: boolean
}>()

const { t } = useI18n()

const form = reactive({
  title: '',
  description: '',
  status: false,
})

function reset() {
  form.title = ''
  form.description = ''
  form.status = false
}

function handleSubmit() {
  const title = form.title.trim()

  if (!title) {
    Message.warning(t('feedback.titleRequired'))
    return
  }

  emit('submit', {
    title,
    description: form.description.trim(),
    status: form.status,
  })
}

defineExpose({
  reset,
})
</script>

<template>
  <section class="task-composer">
    <header class="task-composer__header">
      <div>
        <div class="task-composer__eyebrow">{{ t('tasks.create') }}</div>
        <h2 class="task-composer__title">{{ t('tasks.title') }}</h2>
      </div>
      <p class="task-composer__hint">{{ t('tasks.subtitle') }}</p>
    </header>

    <div class="task-composer__body">
      <label class="task-composer__field">
        <span>{{ t('tasks.titleLabel') }}</span>
        <a-input
          v-model="form.title"
          :placeholder="t('tasks.titleLabel')"
          :max-length="20"
          show-word-limit
          allow-clear
        />
      </label>

      <label class="task-composer__field">
        <span>{{ t('tasks.descriptionLabel') }}</span>
        <a-textarea
          v-model="form.description"
          :placeholder="t('tasks.descriptionLabel')"
          :auto-size="{ minRows: 3, maxRows: 5 }"
          allow-clear
        />
      </label>

      <label class="task-composer__switch">
        <span>{{ t('tasks.statusLabel') }}</span>
        <a-switch v-model="form.status" />
      </label>
    </div>

    <div class="task-composer__footer">
      <a-button type="primary" :loading="loading" @click="handleSubmit">
        {{ t('tasks.create') }}
      </a-button>
    </div>
  </section>
</template>

<style scoped>
.task-composer {
  display: grid;
  gap: 18px;
  padding: 20px;
  border: 1px solid var(--app-line);
  border-radius: var(--app-radius);
  background: var(--app-surface);
}

.task-composer__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: start;
}

.task-composer__eyebrow {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--app-muted);
}

.task-composer__title {
  margin: 4px 0 0;
  font-size: 18px;
}

.task-composer__hint {
  margin: 0;
  max-width: 38ch;
  color: var(--app-muted);
  line-height: 1.6;
}

.task-composer__body {
  display: grid;
  gap: 16px;
}

.task-composer__field,
.task-composer__switch {
  display: grid;
  gap: 8px;
}

.task-composer__field > span,
.task-composer__switch > span {
  font-size: 13px;
  color: var(--app-muted);
}

.task-composer__footer {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 640px) {
  .task-composer__header {
    flex-direction: column;
  }
}
</style>
