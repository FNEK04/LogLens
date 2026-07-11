import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'Home', component: () => import('./views/Home.vue') },
    { path: '/import', name: 'Import', component: () => import('./views/Import.vue') },
    { path: '/query', name: 'Query', component: () => import('./views/Query.vue') },
    { path: '/:pathMatch(.*)*', name: 'NotFound', redirect: '/' },
  ]
})

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

app.mount('#app')
