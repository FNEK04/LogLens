import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

// Import Naive UI
import NaiveUI from 'naive-ui'

// Import components
import Home from './views/Home.vue'
import Import from './views/Import.vue'
import Query from './views/Query.vue'

// Create router
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'Home', component: Home },
    { path: '/import', name: 'Import', component: Import },
    { path: '/query', name: 'Query', component: Query },
  ]
})

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(NaiveUI)

app.mount('#app')
