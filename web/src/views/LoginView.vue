<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const form = reactive({
  email: '',
  password: '',
})

async function submit(mode: 'login' | 'register') {
  try {
    if (mode === 'login') {
      await auth.login(form)
      Message.success(t('feedback.loginSuccess'))
    } else {
      await auth.register(form)
      Message.success(t('feedback.registerSuccess'))
    }

    await router.push({ name: 'tasks' })
  } catch (error) {
    Message.error(error instanceof Error ? error.message : 'Login failed')
  }
}
</script>

<template>
  <main class="auth-page">
    <section class="auth-page__intro">
      <div class="auth-page__mark">{{ t('app.name') }}</div>
      <h1 class="auth-page__title">{{ t('auth.title') }}</h1>
      <p class="auth-page__subtitle">{{ t('auth.subtitle') }}</p>

      <div class="auth-page__notes">
        <div class="auth-page__note">
          <strong>01</strong>
          <span>{{ t('tasks.subtitle') }}</span>
        </div>
        <div class="auth-page__note">
          <strong>02</strong>
          <span>{{ t('app.subtitle') }}</span>
        </div>
        <div class="auth-page__note">
          <strong>03</strong>
          <span>{{ t('nav.locale') }}</span>
        </div>
      </div>
    </section>

    <section class="auth-page__panel">
      <div class="auth-card">
        <div class="auth-card__header">
          <div class="auth-card__eyebrow">{{ t('nav.login') }}</div>
          <h2 class="auth-card__title">{{ t('auth.title') }}</h2>
          <p class="auth-card__hint">{{ t('auth.empty') }}</p>
        </div>

        <div class="auth-card__body">
          <label class="auth-card__field">
            <span>{{ t('auth.email') }}</span>
            <a-input v-model="form.email" :placeholder="t('auth.email')" />
          </label>

          <label class="auth-card__field">
            <span>{{ t('auth.password') }}</span>
            <a-input-password v-model="form.password" :placeholder="t('auth.password')" />
          </label>
        </div>

        <div class="auth-card__actions">
          <a-button type="primary" :loading="auth.busy" @click="submit('login')">
            {{ t('auth.login') }}
          </a-button>
          <a-button :loading="auth.busy" @click="submit('register')">
            {{ t('auth.register') }}
          </a-button>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.auth-page {
  display: grid;
  min-height: 100vh;
  grid-template-columns: minmax(0, 1.1fr) minmax(360px, 460px);
  background: var(--app-bg);
}

.auth-page__intro {
  display: grid;
  align-content: center;
  gap: 20px;
  padding: 56px;
  color: #f8fbff;
  background: linear-gradient(180deg, var(--app-rail), var(--app-rail-soft));
}

.auth-page__mark {
  width: fit-content;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(45, 212, 191, 0.16);
  color: #bff7ec;
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.auth-page__title {
  margin: 0;
  max-width: 12ch;
  font-size: clamp(40px, 5vw, 64px);
  line-height: 0.98;
}

.auth-page__subtitle {
  margin: 0;
  max-width: 42ch;
  font-size: 16px;
  line-height: 1.7;
  color: rgba(248, 251, 255, 0.78);
}

.auth-page__notes {
  display: grid;
  gap: 12px;
  max-width: 480px;
}

.auth-page__note {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 14px 16px;
  border-radius: var(--app-radius);
  background: rgba(255, 255, 255, 0.05);
}

.auth-page__note strong {
  color: var(--app-accent);
}

.auth-page__panel {
  display: grid;
  align-content: center;
  justify-items: center;
  padding: 40px;
}

.auth-card {
  width: min(100%, 420px);
  padding: 24px;
  border: 1px solid var(--app-line);
  border-radius: var(--app-radius);
  background: var(--app-surface);
  box-shadow: 0 16px 32px rgba(16, 32, 51, 0.08);
}

.auth-card__header {
  display: grid;
  gap: 8px;
  margin-bottom: 20px;
}

.auth-card__eyebrow {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--app-muted);
}

.auth-card__title {
  margin: 0;
  font-size: 22px;
}

.auth-card__hint {
  margin: 0;
  color: var(--app-muted);
  line-height: 1.6;
}

.auth-card__body {
  display: grid;
  gap: 16px;
}

.auth-card__field {
  display: grid;
  gap: 8px;
}

.auth-card__field > span {
  font-size: 13px;
  color: var(--app-muted);
}

.auth-card__actions {
  display: grid;
  gap: 12px;
  margin-top: 20px;
}

@media (max-width: 960px) {
  .auth-page {
    grid-template-columns: 1fr;
  }

  .auth-page__intro,
  .auth-page__panel {
    padding: 24px;
  }
}
</style>

