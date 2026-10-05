import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { apiRequest } from '@/api/http'
import { useAuthStore } from './auth'
import type { CreateTaskPayload, Task, TaskListResponse } from '@/api/types'

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref<Task[]>([])
  const total = shallowRef(0)
  const loading = shallowRef(false)
  const submitting = shallowRef(false)
  const lastError = shallowRef<string | null>(null)

  function upsert(task: Task) {
    const exists = tasks.value.some((item) => item.id === task.id)
    tasks.value = [task, ...tasks.value.filter((item) => item.id !== task.id)]

    if (!exists) {
      total.value += 1
    }
  }

  async function loadTasks() {
    const auth = useAuthStore()
    loading.value = true
    lastError.value = null

    try {
      const data = await apiRequest<TaskListResponse>('/api/v1/tasks?pageNo=1&pageSize=20', {
        method: 'GET',
        token: auth.token,
      })

      tasks.value = data.list
      total.value = data.total
    } catch (error) {
      lastError.value = error instanceof Error ? error.message : 'Failed to load tasks'
      throw error
    } finally {
      loading.value = false
    }
  }

  async function createTask(payload: CreateTaskPayload) {
    const auth = useAuthStore()
    submitting.value = true

    try {
      const task = await apiRequest<Task>('/api/v1/tasks', {
        method: 'POST',
        body: JSON.stringify(payload),
        token: auth.token,
      })

      return task
    } finally {
      submitting.value = false
    }
  }

  function remove (id: number) {
    const exists = tasks.value.some((item) => item.id === id)
    if (exists) { total.value -= 1 }
    tasks.value = tasks.value.filter((item) => item.id !== id)
  }

  return {
    createTask,
    lastError,
    loadTasks,
    loading,
    submitting,
    tasks,
    total,
    remove,
    upsert,
  }
})

