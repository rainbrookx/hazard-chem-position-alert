import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '@/api/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/terminals',
      name: 'terminals',
      component: () => import('@/views/TerminalView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/fences',
      name: 'fences',
      component: () => import('@/views/FenceView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/alerts',
      name: 'alerts',
      component: () => import('@/views/AlertView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/',
      redirect: '/terminals',
    },
  ],
})

router.beforeEach((to, _from, next) => {
  if (to.meta.requiresAuth && !getToken()) {
    next('/login')
  } else if (to.path === '/login' && getToken()) {
    next('/terminals')
  } else {
    next()
  }
})

export default router
