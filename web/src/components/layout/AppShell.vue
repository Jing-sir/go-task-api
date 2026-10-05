<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { LOCALE_KEY } from '@/i18n'

const auth = useAuthStore()
const router = useRouter()
const { t, locale } = useI18n({ useScope: 'global' })

const localeLabel = computed(() =>
  locale.value === 'zh-CN' ? 'EN' : '中文',
)

function toggleLocale() {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(LOCALE_KEY, locale.value)
  }
}

function handleLogout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="app-shell">
    <aside class="app-shell__rail">
      <div class="app-shell__brand">
        <div class="app-shell__name">{{ t('app.name') }}</div>
        <div class="app-shell__subtitle">{{ t('app.subtitle') }}</div>
      </div>

      <nav class="app-shell__nav">
        <RouterLink class="app-shell__nav-link" active-class="is-active" to="/tasks">
          {{ t('nav.tasks') }}
        </RouterLink>
      </nav>

      <div class="app-shell__status">
        <div class="app-shell__status-label">{{ t('nav.locale') }}</div>
        <div class="app-shell__status-value">{{ locale }}</div>
      </div>

      <div class="app-shell__user">
        <div class="app-shell__status-label">{{ auth.user?.email || 'guest' }}</div>
        <div class="app-shell__status-value">{{ auth.isLoggedIn ? 'online' : 'offline' }}</div>
      </div>

      <div class="app-shell__actions">
        <a-button block @click="toggleLocale">
          {{ localeLabel }}
        </a-button>
        <a-button block status="danger" @click="handleLogout">
          {{ t('nav.logout') }}
        </a-button>
      </div>
    </aside>

    <main class="app-shell__main">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  display: grid;
  min-height: 100vh;
  grid-template-columns: 280px minmax(0, 1fr);
}

.app-shell__rail {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 28px 22px;
  background: linear-gradient(180deg, var(--app-rail), var(--app-rail-soft));
  color: #f8fbff;
}

.app-shell__brand {
  display: grid;
  gap: 6px;
}

.app-shell__name {
  font-size: 18px;
  font-weight: 700;
}

.app-shell__subtitle {
  color: rgba(248, 251, 255, 0.7);
  line-height: 1.5;
  font-size: 13px;
}

.app-shell__nav {
  display: grid;
  gap: 8px;
}

.app-shell__nav-link {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius);
  color: rgba(248, 251, 255, 0.82);
  background: rgba(255, 255, 255, 0.04);
}

.app-shell__nav-link.is-active {
  background: rgba(45, 212, 191, 0.18);
  color: #fff;
}

.app-shell__status,
.app-shell__user {
  display: grid;
  gap: 4px;
  padding: 12px;
  border-radius: var(--app-radius);
  background: rgba(255, 255, 255, 0.05);
}

.app-shell__status-label {
  font-size: 12px;
  color: rgba(248, 251, 255, 0.68);
}

.app-shell__status-value {
  font-size: 14px;
  font-weight: 600;
  word-break: break-word;
}

.app-shell__actions {
  display: grid;
  gap: 10px;
  margin-top: auto;
}

.app-shell__main {
  min-width: 0;
  padding: 28px;
}

@media (max-width: 960px) {
  .app-shell {
    grid-template-columns: 1fr;
  }

  .app-shell__rail {
    padding: 20px;
  }

  .app-shell__main {
    padding: 20px;
  }
}
</style>
