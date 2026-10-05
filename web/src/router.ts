import type { Pinia } from 'pinia'
import { createRouter as createVueRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import LoginView from '@/views/LoginView.vue'
import TasksView from '@/views/TasksView.vue'

export function createRouter(pinia: Pinia) {
  const router = createVueRouter({
    history: createWebHistory(),
    routes: [
      {
        path: '/',
        redirect: '/tasks',
      },
      {
        path: '/login',
        name: 'login',
        component: LoginView,
      },
      {
        path: '/tasks',
        name: 'tasks',
        component: TasksView,
        meta: {
          requiresAuth: true,
        },
      },
    ],
  })

  router.beforeEach((to) => {
    const auth = useAuthStore(pinia)

    if (to.meta.requiresAuth && !auth.isLoggedIn) {
      return { name: 'login' }
    }

    if (to.name === 'login' && auth.isLoggedIn) {
      return { name: 'tasks' }
    }

    return true
  })

  return router
}

