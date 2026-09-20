<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header">
      <div>
        <el-button text @click="$router.push('/equipment')"><el-icon><ArrowLeft /></el-icon> 返回</el-button>
        <span class="page-title" style="margin-left: 8px">{{ detail?.name }} - 设备详情</span>
      </div>
      <el-button type="primary" :loading="saving" @click="saveInfo">保存</el-button>
    </div>

    <el-row :gutter="16">
      <!-- 左侧：编辑表单 -->
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header><span>基本信息</span></template>
          <el-form ref="formRef" :model="form" :rules="rules" label-width="auto" label-position="left" :require-asterisk-position="'right'">
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="设备名称" prop="name">
                  <el-input v-model="form.name" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="出厂编号" prop="serialNumber">
                  <el-input v-model="form.serialNumber" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="所属检测站">
                  <el-input v-model="form.department" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="所属检测线">
                  <el-input v-model="form.line" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="设备型号">
                  <el-input v-model="form.model" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="厂家">
                  <el-input v-model="form.manufacturer" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="状态">
              <el-select v-model="form.status" style="width:100%">
                <el-option label="使用中" value="in_use" /><el-option label="闲置" value="idle" />
                <el-option label="送检中" value="inspecting" />
              </el-select>
            </el-form-item>
            <el-form-item label="检定证书编号">
              <el-input v-model="form.inspectionCertNumber" style="width:280px" />
            </el-form-item>
            <el-form-item label="检定时间">
              <el-date-picker v-model="form.inspectionDate" type="date" value-format="YYYY-MM-DD" style="width:280px" />
            </el-form-item>
            <el-form-item label="检定有效期">
              <div style="display:flex;align-items:center;gap:4px">
                <el-date-picker v-model="form.inspectionExpiryDate" type="date" value-format="YYYY-MM-DD" style="width:200px" :disabled="permFlag" placeholder="选择日期" />
                <el-checkbox v-model="permFlag" style="white-space:nowrap" @change="onPermToggle">长期有效</el-checkbox>
              </div>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.remark" type="textarea" :rows="2" />
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 右侧：证书管理 -->
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header>
            <div class="cert-header">
              <span>证书信息 ({{ detail?.certificates?.length || 0 }})</span>
              <el-button type="primary" size="small" @click="openCertDialog()">添加证书</el-button>
            </div>
          </template>
          <el-table :data="sortedCerts" stripe border @row-dblclick="(row: any) => previewCert(row)" @sort-change="onCertSort">
            <el-table-column prop="certType" label="类型" min-width="100" sortable="custom" />
            <el-table-column prop="expiryDate" label="到期日期" width="105" sortable="custom">
              <template #default="{ row }">{{ row.expiryDate ? formatDate(row.expiryDate) : '长期有效' }}</template>
            </el-table-column>
            <el-table-column prop="expiryDate" label="状态" width="90" sortable="custom">
              <template #default="{ row }"><ExpiryBadge :expiry-date="row.expiryDate" /></template>
            </el-table-column>
            <el-table-column label="操作" width="170" align="center">
              <template #default="{ row }">
                <el-button text size="small" type="primary" @click="previewCert(row)">预览</el-button>
                <el-button text size="small" type="primary" @click="openCertDialog(row)">编辑</el-button>
                <el-button text size="small" type="danger" @click="removeCert(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加 / 编辑证书对话框 -->
    <el-dialog v-model="certDialog" :title="editingCertId ? '编辑证书' : '添加证书'" :width="isMobile ? '94%' : '520px'" @closed="resetCertForm">
      <el-form ref="certFormRef" :model="certForm" :rules="certRules" label-width="80px">
        <el-form-item label="证书类型" prop="certType">
          <el-input v-model="certForm.certType" placeholder="如：计量检定、校准证书" />
        </el-form-item>
        <el-form-item label="证书编号"><el-input v-model="certForm.certNumber" /></el-form-item>
        <el-form-item label="到期日期">
          <div style="display:flex;align-items:center;gap:4px">
            <el-date-picker v-model="certForm.expiryDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="certPermFlag" placeholder="选择日期" />
            <el-checkbox v-model="certPermFlag" style="white-space:nowrap" @change="onCertPerm">长期有效</el-checkbox>
          </div>
        </el-form-item>
        <el-form-item label="证书文件" prop="fileId">
          <FileUploader @success="(f: any) => (certForm.fileId = f.id)" />
          <el-tag v-if="certForm.fileId" type="success" size="small" closable @close="certForm.fileId = 0" style="margin-top:4px">已上传</el-tag>
          <el-button v-if="certForm.fileId" text size="small" type="danger" @click="certForm.fileId = 0" style="margin-top:4px">删除</el-button>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="certForm.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="certDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingCert" @click="saveCert">保存</el-button>
      </template>
    </el-dialog>

    <!-- 文件预览弹窗 -->
    <el-dialog v-model="previewDialog" title="文件预览" :width="isMobile ? '94%' : '800px'" destroy-on-close>
      <FilePreview v-if="currentFile" :file-id="currentFile.id" :file-name="currentFile.originalName" :mime-type="currentFile.mimeType" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { equipmentApi } from '../../api/equipment';
import { useAuthStore } from '../../stores/auth';
import ExpiryBadge from '../../components/ExpiryBadge.vue';
import FileUploader from '../../components/FileUploader.vue';
import FilePreview from '../../components/FilePreview.vue';
import dayjs from 'dayjs';

const route = useRoute();
const authStore = useAuthStore();
const id = Number(route.params.id);
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const detail = ref<any>(null);
const loading = ref(false);
const saving = ref(false);

// ===== 基本信息表单 =====
const formRef = ref<FormInstance>();
const form = reactive<any>({ name: '', serialNumber: '', department: '', line: '', model: '', manufacturer: '', status: 'in_use', remark: '', inspectionCertNumber: '', inspectionDate: '', inspectionExpiryDate: '' });
const permFlag = ref(false);
function onPermToggle(v: boolean) { if (v) form.inspectionExpiryDate = ''; }
const rules: FormRules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  serialNumber: [{ required: true, message: '请输入出厂编号', trigger: 'blur' }],
};

function fillForm(d: any) {
  Object.assign(form, {
    name: d.name || '',
    serialNumber: d.serialNumber || '',
    department: d.department || '',
    line: d.line || '',
    model: d.model || '',
    manufacturer: d.manufacturer || '',
    status: d.status || 'in_use',
    remark: d.remark || '',
    inspectionCertNumber: d.inspectionCertNumber || '',
    inspectionDate: d.inspectionDate ? d.inspectionDate.substring(0, 10) : '',
    inspectionExpiryDate: d.inspectionExpiryDate ? d.inspectionExpiryDate.substring(0, 10) : '',
  });
  permFlag.value = !d.inspectionExpiryDate;
}

async function saveInfo() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    await equipmentApi.update(id, form);
    ElMessage.success('保存成功');
    loadData();
  } finally { saving.value = false; }
}

// ===== 证书管理 =====
const certDialog = ref(false);
const savingCert = ref(false);
const certFormRef = ref<FormInstance>();
const certForm = reactive<any>({ certType: '', certNumber: '', expiryDate: '', fileId: 0, remark: '' });
const certPermFlag = ref(false);
function onCertPerm(v: boolean) { if (v) certForm.expiryDate = ''; }
// 非空表示当前是「编辑证书」，为空表示「添加证书」
const editingCertId = ref<number | null>(null);
const emptyCertForm = { certType: '', certNumber: '', expiryDate: '', fileId: 0, remark: '' };
function resetCertForm() {
  editingCertId.value = null;
  Object.assign(certForm, emptyCertForm);
  certPermFlag.value = false;
  certFormRef.value?.clearValidate();
}
// 传 row 进入编辑（回填已有证书），不传则新增
function openCertDialog(row?: any) {
  resetCertForm();
  if (row) {
    editingCertId.value = row.id;
    Object.assign(certForm, {
      certType: row.certType || '',
      certNumber: row.certNumber || '',
      expiryDate: row.expiryDate || '',
      fileId: row.fileId || 0,
      remark: row.remark || '',
    });
    certPermFlag.value = !row.expiryDate;
  }
  certDialog.value = true;
}
const certRules: FormRules = {
  certType: [{ required: true, message: '请输入证书类型', trigger: 'blur' }],
  // 编辑时允许沿用已上传的文件（fileId 已回填），新增时必须上传
  fileId: [{ validator: (_r, v, cb) => (v || editingCertId.value ? cb() : cb(new Error('请上传证书文件'))) }],
};

// 证书排序
const certSortProp = ref('expiryDate');
const certSortOrder = ref<'ascending' | 'descending'>('ascending');

const sortedCerts = computed(() => {
  const list = [...(detail.value?.certificates || [])];
  list.sort((a: any, b: any) => {
    let va = a[certSortProp.value];
    let vb = b[certSortProp.value];
    if (va == null) va = '';
    if (vb == null) vb = '';
    const cmp = String(va).localeCompare(String(vb), 'zh-CN');
    return certSortOrder.value === 'ascending' ? cmp : -cmp;
  });
  return list;
});

function onCertSort({ prop, order }: any) {
  if (!order) return;
  certSortProp.value = prop;
  certSortOrder.value = order;
}

async function saveCert() {
  const valid = await certFormRef.value?.validate().catch(() => false);
  if (!valid) return;
  savingCert.value = true;
  try {
    // fileId 为 0 表示没有文件，必须发 null：发 0 会因外键约束（file_id → uploaded_files.id）失败
    const payload = { ...certForm, fileId: certForm.fileId || null };
    if (editingCertId.value) {
      await equipmentApi.updateCert(editingCertId.value, payload);
      ElMessage.success('证书修改成功');
    } else {
      await equipmentApi.addCert(id, payload);
      ElMessage.success('证书添加成功');
    }
    certDialog.value = false;
    loadData();
  } finally { savingCert.value = false; }
}

async function removeCert(row: any) {
  await ElMessageBox.confirm('确认删除该证书？', '警告', { type: 'warning' });
  await equipmentApi.removeCert(row.id);
  ElMessage.success('删除成功');
  loadData();
}

// ===== 文件预览 =====
const previewDialog = ref(false);
const currentFile = ref<any>(null);
function previewCert(row: any) { currentFile.value = row.file; previewDialog.value = true; }

function formatDate(d?: string) { return d ? dayjs(d).format('YYYY-MM-DD') : '-'; }

async function loadData() {
  loading.value = true;
  try {
    const res: any = await equipmentApi.detail(id);
    detail.value = res.data;
    fillForm(res.data);
  } finally { loading.value = false; }
}

onMounted(loadData);
</script>

<style scoped lang="scss">
.cert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
