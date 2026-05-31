import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/AdminLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/DashboardView.vue'),
        meta: { title: '总览' },
      },
      {
        path: 'agents',
        name: 'Agents',
        component: () => import('@/views/AgentListView.vue'),
        meta: { title: '智能体' },
      },
      {
        path: 'agents/:agentId',
        name: 'AgentDetail',
        component: () => import('@/views/AgentDetailView.vue'),
        meta: { title: '智能体详情' },
      },
      {
        path: 'agents/:agentId/versions',
        name: 'AgentVersions',
        component: () => import('@/views/AgentVersionsView.vue'),
        meta: { title: '版本管理' },
      },
      {
        path: 'knowledge',
        name: 'Knowledge',
        component: () => import('@/views/KnowledgeView.vue'),
        meta: { title: '知识库' },
      },
      {
        path: 'genealogy',
        name: 'Genealogy',
        component: () => import('@/views/GenealogyView.vue'),
        meta: { title: '族谱图' },
      },
      {
        path: 'channels',
        name: 'Channels',
        component: () => import('@/views/ChannelView.vue'),
        meta: { title: '渠道接入' },
      },
      {
        path: 'licenses',
        name: 'Licenses',
        component: () => import('@/views/LicenseView.vue'),
        meta: { title: '授权' },
      },
      {
        path: 'analytics',
        name: 'Analytics',
        component: () => import('@/views/AnalyticsView.vue'),
        meta: { title: '数据分析' },
      },
      {
        path: 'webhooks',
        name: 'Webhooks',
        component: () => import('@/views/WebhookView.vue'),
        meta: { title: 'Webhook' },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/SettingsView.vue'),
        meta: { title: '设置' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory('/'),
  routes,
})

// 路由守卫：未登录跳转登录页
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    next('/login')
  } else {
    next()
  }
  // 设置页面标题
  document.title = `${to.meta.title || '管理台'} - 智能体族谱`
})

export default router
