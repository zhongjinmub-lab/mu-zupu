import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', component: () => import('./views/Login.vue') },
  { path: '/dashboard', component: () => import('./views/Dashboard.vue'), meta: { requiresAuth: true } },
  { path: '/agents', component: () => import('./views/Agents.vue'), meta: { requiresAuth: true } },
  { path: '/kbs', component: () => import('./views/KnowledgeBases.vue'), meta: { requiresAuth: true } },
  { path: '/workflows', component: () => import('./views/Workflows.vue'), meta: { requiresAuth: true } },
  { path: '/channels', component: () => import('./views/Channels.vue'), meta: { requiresAuth: true } },
  { path: '/plugins', component: () => import('./views/Plugins.vue'), meta: { requiresAuth: true } },
  { path: '/billing', component: () => import('./views/Billing.vue'), meta: { requiresAuth: true } },
  { path: '/settings', component: () => import('./views/Settings.vue'), meta: { requiresAuth: true } },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    return '/login'
  }
})

export default router
