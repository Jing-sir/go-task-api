import { createApp } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'
import { createPinia } from 'pinia'
import { createRouter } from './router'
import { i18n } from './i18n'
import App from './App.vue'
import './styles/global.css'

const app = createApp(App)
const pinia = createPinia()
const router = createRouter(pinia)

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(ArcoVue)
app.mount('#app')

