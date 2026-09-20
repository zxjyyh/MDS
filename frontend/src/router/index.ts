import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/login/LoginView.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/',
    component: () => import('../layouts/DefaultLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/dashboard/DashboardView.vue'),
        meta: { title: '仪表盘', icon: 'Odometer' },
      },
      {
        path: 'personnel',
        name: 'PersonnelList',
        component: () => import('../views/personnel/PersonnelListView.vue'),
        meta: { title: '人员管理', icon: 'User' },
      },
      {
        path: 'personnel/:id',
        name: 'PersonnelDetail',
        component: () => import('../views/personnel/PersonnelDetailView.vue'),
        meta: { title: '人员详情', hidden: true },
      },
      {
        path: 'equipment',
        name: 'EquipmentList',
        component: () => import('../views/equipment/EquipmentListView.vue'),
        meta: { title: '设备管理', icon: 'Setting' },
      },
      {
        path: 'equipment/:id',
        name: 'EquipmentDetail',
        component: () => import('../views/equipment/EquipmentDetailView.vue'),
        meta: { title: '设备详情', hidden: true },
      },
      {
        path: 'materials/personnel',
        name: 'MaterialsPersonnel',
        component: () => import('../views/materials/MaterialsPersonnelView.vue'),
        meta: { title: '人员材料', icon: 'User' },
      },
      {
        path: 'materials/equipment',
        name: 'MaterialsEquipment',
        component: () => import('../views/materials/MaterialsEquipmentView.vue'),
        meta: { title: '设备材料', icon: 'Setting' },
      },
      {
        path: 'company',
        name: 'CompanyInfo',
        component: () => import('../views/company/CompanyInfoView.vue'),
        // 普通用户不显示该模块（菜单已隐藏，路由同步拦一道防手输 URL）
        meta: { title: '公司信息管理', icon: 'OfficeBuilding', requireAdmin: true },
      },
      {
        path: 'archive/company',
        name: 'CompanyArchiveList',
        component: () => import('../views/archive/CompanyArchiveListView.vue'),
        meta: { title: '公司档案', icon: 'Folder' },
      },
      {
        path: 'archive/company/:id',
        name: 'CompanyArchiveDetail',
        component: () => import('../views/archive/ArchiveDocDetailView.vue'),
        meta: { title: '文档详情', hidden: true },
      },
      {
        path: 'archive/standard',
        name: 'LegalStandardList',
        component: () => import('../views/archive/LegalStandardListView.vue'),
        meta: { title: '法规标准', icon: 'Document' },
      },
      {
        path: 'archive/standard/:id',
        name: 'LegalStandardDetail',
        component: () => import('../views/archive/ArchiveDocDetailView.vue'),
        meta: { title: '文档详情', hidden: true },
      },
      {
        path: 'archive/ip',
        name: 'IpInfo',
        component: () => import('../views/archive/IpInfoView.vue'),
        meta: { title: 'IP信息', icon: 'Connection' },
      },
      {
        path: 'qualification',
        name: 'QualDocList',
        component: () => import('../views/qualification/QualDocListView.vue'),
        meta: { title: '资质体系文件', icon: 'Folder' },
      },
      {
        path: 'qualification/:id',
        name: 'QualDocDetail',
        component: () => import('../views/qualification/QualDocDetailView.vue'),
        meta: { title: '文档详情', hidden: true },
      },
      {
        path: 'safety',
        name: 'SafetyDocList',
        component: () => import('../views/safety/SafetyDocListView.vue'),
        meta: { title: '安全生产体系文件', icon: 'Document' },
      },
      {
        path: 'safety/:id',
        name: 'SafetyDocDetail',
        component: () => import('../views/safety/SafetyDocDetailView.vue'),
        meta: { title: '文档详情', hidden: true },
      },
      {
        path: 'system/users',
        name: 'UserList',
        component: () => import('../views/system/UserListView.vue'),
        meta: { title: '用户管理', icon: 'UserFilled', requireAdmin: true },
      },
      {
        path: 'system/reminders',
        name: 'ReminderLogs',
        component: () => import('../views/system/ReminderLogView.vue'),
        meta: { title: '提醒日志', icon: 'Bell', requireAdmin: true },
      },
      {
        path: 'system/reminder-settings',
        name: 'ReminderSettings',
        component: () => import('../views/system/ReminderSettingsView.vue'),
        meta: { title: '提醒设置', icon: 'AlarmClock', requireAdmin: true },
      },
      {
        path: 'system/operation-logs',
        name: 'OperationLogs',
        component: () => import('../views/system/OperationLogView.vue'),
        meta: { title: '操作日志', icon: 'Tickets', requireAdmin: true },
      },
      {
        path: 'system/backup',
        name: 'Backup',
        component: () => import('../views/system/BackupView.vue'),
        meta: { title: '数据备份', icon: 'FolderOpened', requireAdmin: true },
      },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore();
  if (to.meta.public) {
    if (authStore.isLoggedIn && to.name === 'Login') {
      return next('/dashboard');
    }
    return next();
  }
  if (!authStore.isLoggedIn) {
    return next('/login');
  }
  if (!authStore.user) {
    await authStore.fetchProfile();
  }
  if (to.meta.requireAdmin && !authStore.isAdmin) {
    return next('/dashboard');
  }
  document.title = `${to.meta.title || ''} - 内业管理系统`;
  next();
});

export default router;
