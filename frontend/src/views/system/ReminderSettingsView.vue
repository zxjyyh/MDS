<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">提醒设置</span>
      <el-button type="primary" @click="openAdd">
        <el-icon><Plus /></el-icon>新增规则
      </el-button>
    </div>

    <el-alert type="info" :closable="false" style="margin-bottom: 16px">
      系统每天 08:30 自动检查到期情况，按规则提前天数向管理员邮箱发送提醒邮件。
    </el-alert>

    <el-table :data="rules" v-loading="loading" stripe border>
      <el-table-column type="index" label="#" width="50" />
      <el-table-column label="模块" width="140">
        <template #default="{ row }">
          <el-tag :type="moduleTagType(row.module)" size="small" effect="light">
            {{ moduleLabel(row.module) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="提前天数" width="140" align="center">
        <template #default="{ row }">
          <span style="font-weight: 600; font-size: 16px">{{ row.daysBefore }}</span>
          <span style="color: #909399; margin-left: 4px">天</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-switch
            v-model="row.enabled"
            active-text="启用"
            inactive-text="禁用"
            inline-prompt
            @change="toggleEnabled(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="说明" min-width="200">
        <template #default="{ row }">
          {{ moduleLabel(row.module) }}到期前 {{ row.daysBefore }} 天发送邮件提醒
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right" align="center">
        <template #default="{ row }">
          <div class="action-group">
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="removeRule(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- SMTP 邮件配置 -->
    <el-card v-if="smtpLoaded" shadow="never" style="margin-top: 20px">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>SMTP 邮件配置</span>
          <el-button v-if="!editingSMTP" type="primary" size="small" @click="startEditSMTP">修改</el-button>
          <div v-else>
            <el-button size="small" @click="cancelEditSMTP">取消</el-button>
            <el-button size="small" type="primary" :loading="savingSMTP" @click="saveSMTP">保存</el-button>
          </div>
        </div>
      </template>

      <!-- 只读模式 -->
      <template v-if="!editingSMTP">
        <el-descriptions :column="isMobile ? 1 : 3" border size="small">
          <el-descriptions-item label="SMTP 服务器">{{ smtpConfig.host || '未配置' }}</el-descriptions-item>
          <el-descriptions-item label="端口">{{ smtpConfig.port || '-' }}</el-descriptions-item>
          <el-descriptions-item label="发件邮箱">{{ smtpConfig.from || '未配置' }}</el-descriptions-item>
          <el-descriptions-item label="邮箱账号">{{ smtpConfig.user || '-' }}</el-descriptions-item>
          <el-descriptions-item label="邮箱密码">{{ smtpConfig.pass || '未设置' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="smtpConfig.host ? 'success' : 'info'" size="small">
              {{ smtpConfig.host ? '已配置' : '未配置 (不发送邮件)' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </template>

      <!-- 编辑模式 -->
      <el-form v-else ref="smtpFormRef" :model="smtpForm" :rules="smtpRules" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="SMTP 服务器" prop="host">
              <el-input v-model="smtpForm.host" placeholder="smtp.qq.com" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="smtpForm.port" :min="1" :max="65535" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="加密方式">
              <el-tag size="small" :type="smtpForm.port === 465 ? 'warning' : 'info'">
                {{ smtpForm.port === 465 ? 'SSL' : smtpForm.port === 587 ? 'STARTTLS' : '自定义' }}
              </el-tag>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="发件邮箱" prop="from">
          <el-input v-model="smtpForm.from" placeholder="xxx@qq.com" />
        </el-form-item>
        <el-form-item label="邮箱账号" prop="user">
          <el-input v-model="smtpForm.user" placeholder="xxx@qq.com" />
        </el-form-item>
        <el-form-item label="邮箱密码/授权码" prop="pass">
          <el-input v-model="smtpForm.pass" type="password" show-password placeholder="QQ邮箱需用授权码，非登录密码" />
        </el-form-item>
      </el-form>
    </el-card>

  <!-- 新增/编辑对话框 -->
  <el-dialog v-model="dialogVisible" :title="editing ? '编辑规则' : '新增规则'" :width="isMobile ? '94%' : '480px'" @close="resetForm">
    <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
      <el-form-item label="提醒模块" prop="module">
        <el-select v-model="form.module" style="width: 100%">
          <el-option label="人员证件" value="personnel_cert" />
          <el-option label="设备检定证书" value="equipment_cert" />
          <el-option label="体系文档" value="document" />
        </el-select>
      </el-form-item>
      <el-form-item label="提前天数" prop="daysBefore">
        <el-input-number v-model="form.daysBefore" :min="1" :max="365" style="width: 100%" />
      </el-form-item>
      <el-form-item label="启用状态">
        <el-switch v-model="form.enabled" active-text="启用" inactive-text="禁用" />
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
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { reminderApi, settingsApi } from '../../api/dashboard';
import { useAuthStore } from '../../stores/auth';

const authStore = useAuthStore();
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const rules = ref<any[]>([]);
const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const editing = ref(false);
const formRef = ref<FormInstance>();
const form = reactive({ module: '', daysBefore: 30, enabled: true, id: undefined as number | undefined });
const formRules: FormRules = {
  module: [{ required: true, message: '请选择模块', trigger: 'change' }],
  daysBefore: [{ required: true, message: '请输入提前天数', trigger: 'blur' }],
};

function moduleLabel(m: string) {
  return { personnel_cert: '人员证件', equipment_cert: '设备检定证书', document: '体系文档' }[m] || m;
}
function moduleTagType(m: string): any {
  return { personnel_cert: 'primary', equipment_cert: 'warning', document: 'success' }[m] || 'info';
}

async function loadData() {
  loading.value = true;
  try {
    const res: any = await reminderApi.listRules();
    rules.value = res.data;
  } catch {
    /* backend not ready */
  } finally {
    loading.value = false;
  }
}

function openAdd() {
  editing.value = false;
  Object.assign(form, { module: '', daysBefore: 30, enabled: true, id: undefined });
  dialogVisible.value = true;
}

function openEdit(row: any) {
  editing.value = true;
  Object.assign(form, { module: row.module, daysBefore: row.daysBefore, enabled: row.enabled, id: row.id });
  dialogVisible.value = true;
}

function resetForm() {
  editing.value = false;
  Object.assign(form, { module: '', daysBefore: 30, enabled: true, id: undefined });
}

async function save() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    if (editing.value) {
      await reminderApi.updateRule(form.id!, { module: form.module, daysBefore: form.daysBefore, enabled: form.enabled });
      ElMessage.success('规则已更新');
    } else {
      await reminderApi.addRule({ module: form.module, daysBefore: form.daysBefore, enabled: form.enabled });
      ElMessage.success('规则已创建');
    }
    dialogVisible.value = false;
    loadData();
  } catch (err: any) {
    if (err.response?.data?.message) ElMessage.error(err.response.data.message);
  } finally { saving.value = false; }
}

async function toggleEnabled(row: any) {
  try {
    await reminderApi.updateRule(row.id, { enabled: row.enabled });
    ElMessage.success(row.enabled ? '规则已启用' : '规则已禁用');
  } catch {
    row.enabled = !row.enabled;
    ElMessage.error('操作失败');
  }
}

async function removeRule(row: any) {
  await ElMessageBox.confirm(
    `确认删除规则：${moduleLabel(row.module)}到期前 ${row.daysBefore} 天提醒？`,
    '警告',
    { type: 'warning' }
  );
  await reminderApi.deleteRule(row.id);
  ElMessage.success('规则已删除');
  loadData();
}

// ===== SMTP 配置 =====
const editingSMTP = ref(false);
const savingSMTP = ref(false);
const smtpFormRef = ref<FormInstance>();
const smtpConfig = reactive({ host: '', port: 465, user: '', pass: '', from: '' });
const smtpForm = reactive({ host: '', port: 465, user: '', pass: '', from: '' });
const smtpLoaded = ref(false);
const smtpRules: FormRules = {
  host: [{ required: true, message: '请输入 SMTP 服务器地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  from: [{ required: true, message: '请输入发件邮箱', trigger: 'blur' }],
  user: [{ required: true, message: '请输入邮箱账号', trigger: 'blur' }],
  pass: [{ required: true, message: '请输入邮箱密码或授权码', trigger: 'blur' }],
};

async function loadSMTP() {
  try {
    const res: any = await settingsApi.getSMTP();
    Object.assign(smtpConfig, res.data);
    smtpLoaded.value = true;
  } catch {
    smtpLoaded.value = false;
  }
}

function startEditSMTP() {
  Object.assign(smtpForm, { ...smtpConfig, pass: '' }); // 编辑时密码字段清空，不填表示不修改
  editingSMTP.value = true;
}

function cancelEditSMTP() {
  editingSMTP.value = false;
}

async function saveSMTP() {
  await smtpFormRef.value?.validate();
  savingSMTP.value = true;
  try {
    const payload = { ...smtpForm };
    if (!payload.pass) {
      delete (payload as any).pass; // 不修改密码
    }
    await settingsApi.updateSMTP(payload);
    ElMessage.success('SMTP 配置已保存');
    editingSMTP.value = false;
    loadSMTP();
  } catch (err: any) {
    if (err.response?.data?.message) ElMessage.error(err.response.data.message);
  } finally { savingSMTP.value = false; }
}

onMounted(() => { loadData(); loadSMTP(); });
</script>

<style scoped lang="scss">
.action-group {
  display: flex;
  gap: 6px;
  justify-content: center;
}
</style>
