<template>
  <el-container class="layout-container">
    <!-- PC端：固定侧边栏 -->
    <el-aside v-if="!isMobile" :width="collapsed ? '64px' : '220px'" class="sidebar">
      <!-- Logo 区 -->
      <div class="logo">
        <el-icon size="24" color="#fff"><OfficeBuilding /></el-icon>
        <transition name="fade-slide">
          <span v-show="!collapsed" class="logo-text">内业管理系统</span>
        </transition>
      </div>

      <!-- 菜单 -->
      <el-scrollbar class="menu-scroll">
        <el-menu
          :default-active="activeMenu"
          :collapse="collapsed"
          :collapse-transition="false"
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409eff"
          router
        >
          <el-menu-item index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-menu-item index="/personnel">
            <el-icon><User /></el-icon>
            <template #title>人员管理</template>
          </el-menu-item>
          <el-menu-item index="/equipment">
            <el-icon><Setting /></el-icon>
            <template #title>设备管理</template>
          </el-menu-item>
          <el-menu-item index="/qualification">
            <el-icon><Folder /></el-icon>
            <template #title>资质体系文件</template>
          </el-menu-item>
          <el-menu-item index="/safety">
            <el-icon><Document /></el-icon>
            <template #title>安全生产文件</template>
          </el-menu-item>
          <el-sub-menu v-if="authStore.isAdmin" index="system">
            <template #title>
              <el-icon><Tools /></el-icon>
              <span>系统管理</span>
            </template>
            <el-menu-item index="/system/users">
              <el-icon><UserFilled /></el-icon>
              <template #title>用户管理</template>
            </el-menu-item>
            <el-menu-item index="/system/reminders">
              <el-icon><Bell /></el-icon>
              <template #title>提醒日志</template>
            </el-menu-item>
            <el-menu-item index="/system/reminder-settings">
              <el-icon><AlarmClock /></el-icon>
              <template #title>提醒设置</template>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-scrollbar>

      <!-- 底部折叠按钮 -->
      <div class="collapse-footer" @click="toggleCollapse">
        <el-icon size="18" color="#bfcbd9">
          <Fold v-if="!collapsed" />
          <Expand v-else />
        </el-icon>
        <transition name="fade-slide">
          <span v-show="!collapsed" class="collapse-text">收起菜单</span>
        </transition>
      </div>
    </el-aside>

    <!-- 移动端：抽屉式侧边栏 -->
    <el-drawer
      v-if="isMobile"
      v-model="drawerVisible"
      direction="ltr"
      :size="240"
      :with-header="false"
      class="mobile-drawer"
    >
      <div class="sidebar drawer-sidebar">
        <div class="logo">
          <el-icon size="24" color="#fff"><OfficeBuilding /></el-icon>
          <span class="logo-text">内业管理系统</span>
        </div>
        <el-scrollbar class="menu-scroll">
          <el-menu
            :default-active="activeMenu"
            background-color="#304156"
            text-color="#bfcbd9"
            active-text-color="#409eff"
            router
            @select="onMenuSelect"
          >
            <el-menu-item index="/dashboard">
              <el-icon><Odometer /></el-icon>
              <template #title>仪表盘</template>
            </el-menu-item>
            <el-menu-item index="/personnel">
              <el-icon><User /></el-icon>
              <template #title>人员管理</template>
            </el-menu-item>
            <el-menu-item index="/equipment">
              <el-icon><Setting /></el-icon>
              <template #title>设备管理</template>
            </el-menu-item>
            <el-menu-item index="/qualification">
              <el-icon><Folder /></el-icon>
              <template #title>资质体系文件</template>
            </el-menu-item>
            <el-menu-item index="/safety">
              <el-icon><Document /></el-icon>
              <template #title>安全生产文件</template>
            </el-menu-item>
            <el-sub-menu v-if="authStore.isAdmin" index="system">
              <template #title>
                <el-icon><Tools /></el-icon>
                <span>系统管理</span>
              </template>
              <el-menu-item index="/system/users">
                <el-icon><UserFilled /></el-icon>
                <template #title>用户管理</template>
              </el-menu-item>
              <el-menu-item index="/system/reminders">
                <el-icon><Bell /></el-icon>
                <template #title>提醒日志</template>
              </el-menu-item>
              <el-menu-item index="/system/reminder-settings">
                <el-icon><AlarmClock /></el-icon>
                <template #title>提醒设置</template>
              </el-menu-item>
            </el-sub-menu>
          </el-menu>
        </el-scrollbar>
      </div>
    </el-drawer>

    <el-container>
      <el-header class="header">
        <div class="header-left">
          <!-- 移动端汉堡菜单 -->
          <el-icon v-if="isMobile" size="22" class="menu-trigger" @click="drawerVisible = true">
            <Expand />
          </el-icon>
          <el-breadcrumb v-if="!isMobile" separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
          <span v-else class="mobile-page-title">{{ currentTitle }}</span>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleCommand" trigger="click">
            <span class="user-info">
              <el-avatar :size="32" icon="UserFilled" />
              <span v-if="!isMobile" class="username">{{ authStore.user?.realName || authStore.user?.username }}</span>
              <el-tag v-if="!isMobile" size="small" :type="authStore.isAdmin ? 'danger' : 'info'">
                {{ authStore.isAdmin ? '管理员' : '普通用户' }}
              </el-tag>
              <el-icon v-if="isMobile" size="16" color="#606266"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>

    <!-- 修改密码对话框 -->
    <el-dialog v-model="pwdDialog" title="修改密码" width="420px">
      <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="80px">
        <el-form-item label="原密码" prop="oldPassword">
          <el-input v-model="pwdForm.oldPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input v-model="pwdForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="pwdForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialog = false">取消</el-button>
        <el-button type="primary" @click="submitPassword">确认</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { useAuthStore } from '../stores/auth';
import { authApi } from '../api/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

// 响应式判断
const isMobile = ref(window.innerWidth <= 768);
const drawerVisible = ref(false);

function handleResize() {
  const wasMobile = isMobile.value;
  isMobile.value = window.innerWidth <= 768;
  // 从手机切到PC时关闭抽屉
  if (wasMobile && !isMobile.value) {
    drawerVisible.value = false;
  }
}

onMounted(() => window.addEventListener('resize', handleResize));
onUnmounted(() => window.removeEventListener('resize', handleResize));

// 从 localStorage 读取折叠状态，默认展开
const collapsed = ref(localStorage.getItem('sidebarCollapsed') === 'true');

function toggleCollapse() {
  collapsed.value = !collapsed.value;
  localStorage.setItem('sidebarCollapsed', String(collapsed.value));
}

// 移动端选择菜单后关闭抽屉
function onMenuSelect() {
  drawerVisible.value = false;
}

const activeMenu = computed(() => route.path);
const currentTitle = computed(() => (route.meta.title as string) || '');

const pwdDialog = ref(false);
const pwdFormRef = ref<FormInstance>();
const pwdForm = ref({ oldPassword: '', newPassword: '', confirmPassword: '' });
const pwdRules: FormRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    {
      validator: (_rule, value, callback) => {
        if (value !== pwdForm.value.newPassword) callback(new Error('两次密码不一致'));
        else callback();
      },
      trigger: 'blur',
    },
  ],
};

function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    ElMessageBox.confirm('确认退出登录？', '提示', { type: 'warning' }).then(() => {
      authStore.logout();
      router.push('/login');
    });
  } else if (cmd === 'password') {
    pwdForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' };
    pwdDialog.value = true;
  }
}

async function submitPassword() {
  await pwdFormRef.value?.validate();
  await authApi.changePassword({
    oldPassword: pwdForm.value.oldPassword,
    newPassword: pwdForm.value.newPassword,
  });
  ElMessage.success('密码修改成功');
  pwdDialog.value = false;
}
</script>

<style scoped lang="scss">
.layout-container {
  height: 100%;
}
.sidebar {
  background: #304156;
  transition: width 0.3s ease;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
}
.drawer-sidebar {
  height: 100%;
  width: 240px;
  transition: none;
}
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #fff;
  flex-shrink: 0;
  border-bottom: 1px solid #1f2d3d;
  .logo-text {
    font-size: 16px;
    font-weight: 600;
    white-space: nowrap;
  }
}
.menu-scroll {
  flex: 1;
  min-height: 0;
  :deep(.el-scrollbar__wrap) {
    background: #304156;
  }
}
/* 底部折叠按钮 */
.collapse-footer {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  border-top: 1px solid #1f2d3d;
  background: #263445;
  transition: background 0.2s;
  flex-shrink: 0;
  &:hover {
    background: #1f2d3d;
  }
  .collapse-text {
    color: #bfcbd9;
    font-size: 13px;
    white-space: nowrap;
  }
}
.header {
  background: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 56px;
  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .menu-trigger {
    cursor: pointer;
    color: #303133;
    &:active {
      opacity: 0.6;
    }
  }
  .mobile-page-title {
    font-size: 17px;
    font-weight: 600;
  }
  .user-info {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    .username {
      font-size: 14px;
    }
  }
}
.main-content {
  background: #f5f7fa;
  padding: 0;
  overflow-y: auto;
  overflow-x: hidden;
  min-width: 0;
}
:deep(.el-menu) {
  border-right: none;
}
/* 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.fade-slide-enter-active {
  transition: all 0.25s ease;
}
.fade-slide-leave-active {
  transition: all 0.15s ease;
}
.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(-10px);
}
.fade-slide-leave-to {
  opacity: 0;
}
</style>

<style lang="scss">
/* 移动端抽屉样式覆盖 */
.mobile-drawer .el-drawer__body {
  padding: 0;
}
</style>
