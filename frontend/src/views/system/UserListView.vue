<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">用户管理</span>
      <el-button type="primary" @click="dialogVisible = true"><el-icon><Plus /></el-icon>新增用户</el-button>
    </div>

    <div class="search-bar">
      <el-input v-model="query.keyword" placeholder="用户名/姓名/邮箱" clearable style="width: 220px" @keyup.enter="loadData" />
      <el-select v-model="query.role" placeholder="角色" clearable style="width: 120px">
        <el-option label="管理员" value="admin" />
        <el-option label="普通用户" value="user" />
      </el-select>
      <el-button type="primary" @click="loadData">查询</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe border>
      <el-table-column type="index" label="#" width="50" />
      <el-table-column prop="username" label="用户名" width="120" />
      <el-table-column prop="realName" label="姓名" width="100" />
      <el-table-column prop="email" label="邮箱" min-width="180" />
      <el-table-column label="角色" width="100">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
            {{ row.role === 'admin' ? '管理员' : '普通用户' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
            {{ row.status === 'active' ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最近登录" width="160">
        <template #default="{ row }">{{ row.lastLoginAt ? formatDateTime(row.lastLoginAt) : '-' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button text size="small" @click="editRow(row)">编辑</el-button>
          <el-button text size="small" @click="resetPwd(row)">重置密码</el-button>
          <el-button text size="small" :type="row.status === 'active' ? 'warning' : 'success'" @click="toggleStatus(row)">
            {{ row.status === 'active' ? '禁用' : '启用' }}
          </el-button>
          <el-button
            text
            size="small"
            type="danger"
            @click="removeRow(row)"
            :disabled="row.username === 'admin' || row.id === authStore.user?.id"
          >删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
    </div>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑用户' : '新增用户'" :width="isMobile ? '94%' : '500px'" @close="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="登录用的用户名" />
          <div v-if="editing" class="form-tip">用户名即登录名，改完后需用新用户名登录</div>
        </el-form-item>
        <el-form-item v-if="!editing" label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="姓名" prop="realName"><el-input v-model="form.realName" /></el-form-item>
        <el-form-item label="邮箱" prop="email"><el-input v-model="form.email" /></el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox, ElMessageBoxOptions, type FormInstance, type FormRules } from 'element-plus';
import { userApi } from '../../api/user';
import { useAuthStore } from '../../stores/auth';
import dayjs from 'dayjs';

const authStore = useAuthStore();
const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive({ page: 1, pageSize: 20, keyword: '', role: '' });
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const dialogVisible = ref(false);
const editing = ref(false);
const saving = ref(false);
const formRef = ref<FormInstance>();
const form = reactive<any>({ username: '', password: '', realName: '', email: '', role: 'user' });
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }, { min: 2, message: '至少2位', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }, { min: 6, message: '至少6位', trigger: 'blur' }],
  realName: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }, { type: 'email', message: '邮箱格式不正确', trigger: 'blur' }],
  role: [{ required: true, message: '请选择角色', trigger: 'change' }],
};

function formatDateTime(d: string) { return dayjs(d).format('YYYY-MM-DD HH:mm'); }

async function loadData() {
  loading.value = true;
  try {
    const res: any = await userApi.list(query);
    list.value = res.data.list;
    total.value = res.data.total;
  } finally { loading.value = false; }
}

function editRow(row: any) {
  editing.value = true;
  Object.assign(form, { ...row, password: '' });
  dialogVisible.value = true;
}

function resetForm() {
  editing.value = false;
  Object.assign(form, { username: '', password: '', realName: '', email: '', role: 'user', id: undefined });
}

async function save() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    if (editing.value) {
      await userApi.update(form.id, {
        username: form.username,
        realName: form.realName,
        email: form.email,
        role: form.role,
      });
      ElMessage.success('更新成功');
    } else {
      await userApi.create(form);
      ElMessage.success('创建成功');
    }
    dialogVisible.value = false;
    loadData();
  } finally { saving.value = false; }
}

async function resetPwd(row: any) {
  const { value } = await ElMessageBox.prompt('请输入新密码（至少6位）', `重置「${row.realName}」的密码`, {
    inputPattern: /^.{6,}$/,
    inputErrorMessage: '密码至少6位',
  });
  await userApi.resetPassword(row.id, value);
  ElMessage.success('密码重置成功');
}

async function toggleStatus(row: any) {
  const newStatus = row.status === 'active' ? 'disabled' : 'active';
  await userApi.setStatus(row.id, newStatus);
  ElMessage.success('状态更新成功');
  loadData();
}

async function removeRow(row: any) {
  await ElMessageBox.confirm(`确认删除用户「${row.realName}」？`, '警告', { type: 'warning' } as ElMessageBoxOptions);
  await userApi.remove(row.id);
  ElMessage.success('删除成功');
  loadData();
}

onMounted(loadData);
</script>

<style scoped lang="scss">
.form-tip {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
</style>
