import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true }
    },
    {
      path: '/',
      component: () => import('@/views/LayoutView.vue'),
      redirect: '/orders',
      children: [
        // 工单管理
        { path: 'orders', component: () => import('@/views/order/OrderList.vue'), meta: { title: '工单列表' } },
        { path: 'orders/:id', component: () => import('@/views/order/OrderDetail.vue'), meta: { title: '工单详情' } },
        // 报工
        { path: 'report-progress', component: () => import('@/views/report/ReportProgress.vue'), meta: { title: '报工进度' } },
        { path: 'report-search', component: () => import('@/views/report/ReportSearch.vue'), meta: { title: '报工查询' } },
        // 工艺管理
        { path: 'processes', component: () => import('@/views/process/ProcessList.vue'), meta: { title: '工序模板' } },
        { path: 'process-aliases', component: () => import('@/views/process/ProcessAlias.vue'), meta: { title: '流程模板' } },
        // 系统管理
        { path: 'users', component: () => import('@/views/system/UserManage.vue'), meta: { title: '用户管理' } },
        { path: 'departments', component: () => import('@/views/system/DeptManage.vue'), meta: { title: '部门管理' } },
        { path: 'roles', component: () => import('@/views/system/RolePermission.vue'), meta: { title: '角色权限' } }
      ]
    }
  ]
})

// 路由守卫
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isLoggedIn) {
    return '/login'
  }
  if (to.path === '/login' && auth.isLoggedIn) {
    return '/orders'
  }
})

export default router

