import { computed, ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { apiRequest } from '@/api/http'
import type { AuthPayload, LoginResponse, User } from '@/api/types'

const STORAGE_KEY = 'go-task-web.session'

interface StoredSession {
  token: string
  user: User
}

function readSession() {
  if (typeof window === 'undefined') {
    return null
  }

  const raw = window.localStorage.getItem(STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as StoredSession
  } catch {
    window.localStorage.removeItem(STORAGE_KEY)
    return null
  }
}

function saveSession(session: StoredSession | null) {
  if (typeof window === 'undefined') {
    return
  }

  if (!session) {
    window.localStorage.removeItem(STORAGE_KEY)
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
}

export const useAuthStore = defineStore('auth', () => {
  const session = readSession()

  const token = shallowRef<string | null>(session?.token ?? null)
  const user = ref<User | null>(session?.user ?? null)
  const busy = shallowRef(false)

  const isLoggedIn = computed(() => Boolean(token.value))

  async function login(payload: AuthPayload) {
    busy.value = true

    try {
      const data = await apiRequest<LoginResponse>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify(payload),
      })

      token.value = data.token
      user.value = data.user
      saveSession({ token: data.token, user: data.user })

      return data
    } finally {
      busy.value = false
    }
  }

  async function register(payload: AuthPayload) {
    busy.value = true

    try {
      await apiRequest<User>('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify(payload),
      })

      return login(payload)
    } finally {
      busy.value = false
    }
  }

  function logout() {
    token.value = null
    user.value = null
    saveSession(null)
  }

  return {
    busy,
    isLoggedIn,
    login,
    logout,
    register,
    token,
    user,
  }
})

