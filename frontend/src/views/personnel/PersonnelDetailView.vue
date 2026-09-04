<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header">
      <div>
        <el-button text @click="$router.push('/personnel')"><el-icon><ArrowLeft /></el-icon> 返回</el-button>
        <span class="page-title" style="margin-left: 8px">{{ detail?.name }} - 人员详情</span>
      </div>
      <el-button v-if="authStore.isAdmin" type="primary" :loading="saving" @click="saveInfo">保存</el-button>
    </div>

    <el-row :gutter="16">
      <!-- 左侧：编辑表单 -->
      <el-col :xs="24" :md="12">
        <el-card shadow="never">
          <template #header><span>基本信息</span></template>
          <el-form ref="formRef" :model="form" :rules="rules" label-width="auto" label-position="left" :require-asterisk-position="'right'">
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="姓名" prop="name">
                  <el-input v-model="form.name" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="电话">
                  <el-input v-model="form.phone" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="性别">
                  <el-select v-model="form.gender" style="width:100%">
                    <el-option label="男" value="male" />
                    <el-option label="女" value="female" />
                    <el-option label="未知" value="unknown" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="职称">
                  <el-input v-model="form.position" style="width:100%" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="身份证号">
                  <el-input v-model="form.idCard" style="width:100%" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="身份证到期">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.idCardExpiry" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.idCard" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.idCard" style="flex-shrink:0;white-space:nowrap" @change="onPermChange('idCardExpiry', permFlags.idCard)">长期有效</el-checkbox>
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="住址">
              <el-input v-model="form.address" style="width:100%;max-width:360px" />
            </el-form-item>
            <el-form-item label="所属检测站">
              <el-input v-model="form.department" style="width:100%;max-width:260px" />
            </el-form-item>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="入职日期">
                  <el-date-picker v-model="form.entryDate" type="date" value-format="YYYY-MM-DD" style="width:100%" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="在职状态">
                  <el-select v-model="form.status" style="width:100%">
                    <el-option label="在职" value="active" />
                    <el-option label="离职" value="resigned" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="学历">
                  <el-select v-model="form.education" style="width:100%" clearable>
                    <el-option label="高中/中专" value="高中/中专" />
                    <el-option label="大专" value="大专" />
                    <el-option label="本科及以上" value="本科及以上" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="毕业学校">
                  <el-input v-model="form.school" style="width:100%" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="专业">
                  <el-input v-model="form.major" style="width:100%" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="毕业时间">
                  <el-date-picker v-model="form.graduationDate" type="date" value-format="YYYY-MM-DD" style="width:100%" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="驾驶证">
                  <el-input v-model="form.driverLicense" placeholder="车型" style="width:150px" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="驾照到期">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.driverLicenseExpiry" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.driverLicense" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.driverLicense" style="flex-shrink:0;white-space:nowrap" @change="onPermChange('driverLicenseExpiry', permFlags.driverLicense)">长期有效</el-checkbox>
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="安检证">
              <el-row :gutter="8" style="width:100%">
                <el-col :span="10"><el-input v-model="form.securityCertNumber" placeholder="编号" /></el-col>
                <el-col :span="14">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.securityCertDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.securityCert" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.securityCert" style="flex-shrink:0" @change="onPermChange('securityCertDate', permFlags.securityCert)">长期有效</el-checkbox>
                  </div>
                </el-col>
              </el-row>
            </el-form-item>
            <el-form-item label="环保证">
              <el-row :gutter="8" style="width:100%">
                <el-col :span="10"><el-input v-model="form.envCertNumber" placeholder="编号" /></el-col>
                <el-col :span="14">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.envCertDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.envCert" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.envCert" style="flex-shrink:0" @change="onPermChange('envCertDate', permFlags.envCert)">长期有效</el-checkbox>
                  </div>
                </el-col>
              </el-row>
            </el-form-item>
            <el-form-item label="查验证">
              <el-row :gutter="8" style="width:100%">
                <el-col :span="10"><el-input v-model="form.inspectionCertNumber" placeholder="编号" /></el-col>
                <el-col :span="14">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.inspectionDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.inspection" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.inspection" style="flex-shrink:0" @change="onPermChange('inspectionDate', permFlags.inspection)">长期有效</el-checkbox>
                  </div>
                </el-col>
              </el-row>
            </el-form-item>
            <el-form-item label="内审员证">
              <el-row :gutter="8" style="width:100%">
                <el-col :span="10"><el-input v-model="form.internalAuditorNumber" placeholder="编号" /></el-col>
                <el-col :span="14">
                  <div style="display:flex;align-items:center;gap:4px">
                    <el-date-picker v-model="form.internalAuditorDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="permFlags.internalAuditor" placeholder="到期时间" />
                    <el-checkbox v-model="permFlags.internalAuditor" style="flex-shrink:0" @change="onPermChange('internalAuditorDate', permFlags.internalAuditor)">长期有效</el-checkbox>
                  </div>
                </el-col>
              </el-row>
            </el-form-item>
            <el-form-item label="员工照片">
              <div class="photo-upload-row">
                <FileUploader accept="image/*" @success="(f: any) => (form.photoFileId = f.id)" />
                <span v-if="form.photoFileId" class="photo-tip">已选择新照片，保存后生效</span>
                <template v-if="detail?.photoFile">
                  <img :src="fileApi.previewUrl(detail.photoFile.id)" class="current-photo" alt="当前照片" />
                  <el-button type="danger" size="small" plain @click="removePhoto">删除照片</el-button>
                </template>
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
              <el-button v-if="authStore.isAdmin" type="primary" size="small" @click="openAddCert">添加证件</el-button>
            </div>
          </template>
          <el-table :data="sortedCerts" stripe border :row-class-name="certRowClass" @row-dblclick="(row: any) => previewCert(row)" @sort-change="onCertSort">
            <el-table-column prop="certName" label="证件名称" min-width="120" show-overflow-tooltip sortable="custom">
              <template #default="{ row }">
                <span>{{ row.certName }}</span>
                <el-tag v-if="isReservedCertName(row.certName)" type="info" size="small" effect="plain" style="margin-left:6px">固定</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="expiryDate" label="到期日期" width="105" sortable="custom">
              <template #default="{ row }">{{ row.fileId ? (row.expiryDate ? formatDate(row.expiryDate) : '长期有效') : '—' }}</template>
            </el-table-column>
            <el-table-column prop="expiryDate" label="状态" width="85" sortable="custom">
              <template #default="{ row }">
                <el-tag v-if="!row.fileId" type="info" size="small" effect="light">待上传</el-tag>
                <ExpiryBadge v-else :expiry-date="row.expiryDate" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="195" align="center">
              <template #default="{ row }">
                <el-button text size="small" type="primary" @click.stop="openEditCert(row)">编辑</el-button>
                <el-button text size="small" type="primary" :disabled="!row.file" @click.stop="previewCert(row)">预览</el-button>
                <el-button v-if="authStore.isAdmin && !isReservedCertName(row.certName)" text size="small" type="danger" @click.stop="removeCert(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加证件对话框 -->
    <el-dialog v-model="certDialog" :title="editingCertId ? '编辑证件' : '添加证件'" :width="isMobile ? '94%' : '520px'">
      <el-form ref="certFormRef" :model="certForm" :rules="certRules" label-width="80px">
        <el-form-item label="证件名称" prop="certName">
          <el-input v-model="certForm.certName" :disabled="certNameLocked" placeholder="如：身份证、学历证书、驾驶证" />
          <div v-if="certNameLocked" class="cert-name-tip">固定预留项，名称不可修改；上传文件并填写编号/到期日期后保存即可</div>
        </el-form-item>
        <el-form-item label="证件编号"><el-input v-model="certForm.certNumber" /></el-form-item>
        <el-form-item label="到期日期">
              <div style="display:flex;align-items:center;gap:4px">
                <el-date-picker v-model="certForm.expiryDate" type="date" value-format="YYYY-MM-DD" style="flex:1" :disabled="certPermFlag" placeholder="选择日期" />
                <el-checkbox v-model="certPermFlag" style="white-space:nowrap" @change="onCertPerm">长期有效</el-checkbox>
              </div>
            </el-form-item>
        <el-form-item label="证件文件" prop="fileId">
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
import { personnelApi } from '../../api/personnel';
import { fileApi } from '../../api/file';
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
const removePhotoFlag = ref(false);
function removePhoto() {
  form.photoFileId = 0;
  removePhotoFlag.value = true;
}

// ===== 基本信息表单 =====
const formRef = ref<FormInstance>();
const form = reactive<any>({
  name: '', gender: 'unknown', idCard: '', phone: '', department: '', address: '', position: '',
  entryDate: '', education: '', school: '', major: '', driverLicense: '',
  driverLicenseExpiry: '', idCardExpiry: '', graduationDate: '',
  securityCertDate: '', envCertDate: '', inspectionDate: '', internalAuditorDate: '',
  securityCertNumber: '', envCertNumber: '', inspectionCertNumber: '', internalAuditorNumber: '',
  status: 'active', remark: '', photoFileId: 0,
});
const rules: FormRules = {
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
};

// 长期有效 标记
const permFlags = reactive({ securityCert: false, envCert: false, inspection: false, internalAuditor: false, driverLicense: false, idCard: false });
function onPermChange(field: string, checked: boolean) {
  if (checked) (form as any)[field] = '';
}

function fillForm(d: any) {
  Object.assign(form, {
    name: d.name,
    gender: d.gender || 'unknown',
    idCard: d.idCard || '',
    phone: d.phone || '',
    department: d.department || '',
    address: d.address || '',
    position: d.position || '',
    entryDate: d.entryDate ? d.entryDate.split('T')[0] : '',
    education: d.education || '',
    school: d.school || '',
    major: d.major || '',
    driverLicense: d.driverLicense || '',
    driverLicenseExpiry: d.driverLicenseExpiry ? d.driverLicenseExpiry.substring(0, 10) : '',
    idCardExpiry: d.idCardExpiry ? d.idCardExpiry.substring(0, 10) : '',
    graduationDate: d.graduationDate ? d.graduationDate.substring(0, 10) : '',
    securityCertDate: d.securityCertDate ? d.securityCertDate.substring(0, 10) : '',
    securityCertNumber: d.securityCertNumber || '',
    envCertDate: d.envCertDate ? d.envCertDate.substring(0, 10) : '',
    envCertNumber: d.envCertNumber || '',
    inspectionDate: d.inspectionDate ? d.inspectionDate.substring(0, 10) : '',
    inspectionCertNumber: d.inspectionCertNumber || '',
    internalAuditorDate: d.internalAuditorDate ? d.internalAuditorDate.substring(0, 10) : '',
    internalAuditorNumber: d.internalAuditorNumber || '',
    status: d.status || 'active',
    remark: d.remark || '',
    photoFileId: d.photoFileId || 0,
  });
  permFlags.securityCert = !d.securityCertDate;
  permFlags.envCert = !d.envCertDate;
  permFlags.inspection = !d.inspectionDate;
  permFlags.internalAuditor = !d.internalAuditorDate;
  permFlags.driverLicense = !d.driverLicenseExpiry;
  permFlags.idCard = !d.idCardExpiry;
}

async function saveInfo() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    const payload = { ...form };
    if (removePhotoFlag.value) payload.photoFileId = null;
    else if (!payload.photoFileId) delete payload.photoFileId;
    await personnelApi.update(id, payload);
    ElMessage.success('保存成功');
    loadData();
  } finally {
    saving.value = false;
  }
}

// ===== 证书管理 =====
// 每位人员固定预留的两个证件（名称即锚点，与后端保持一致；不可改名、不可删除）
const RESERVED_CERT_NAMES = ['安检证', '环保证'];
function isReservedCertName(name?: string) {
  return !!name && RESERVED_CERT_NAMES.includes(name);
}

const certDialog = ref(false);
const savingCert = ref(false);
const editingCertId = ref<number | null>(null);
// 编辑固定预留项时证书名称锁定（禁用输入）
const certNameLocked = ref(false);
const certFormRef = ref<FormInstance>();
const certForm = reactive<any>({ certName: '', certNumber: '', expiryDate: '', fileId: 0, remark: '' });
const certPermFlag = ref(false);
function onCertPerm(v: boolean) { if (v) certForm.expiryDate = ''; }
const certRules: FormRules = {
  certName: [{ required: true, message: '请输入证件名称', trigger: 'blur' }],
  fileId: [{ required: true, message: '请上传证件文件', validator: (_r, v, cb) => (v ? cb() : cb(new Error('请上传证件文件'))) }],
};

// 证书排序
const certSortProp = ref('expiryDate');
const certSortOrder = ref<'ascending' | 'descending'>('ascending');

const sortedCerts = computed(() => {
  const all = detail.value?.certificates || [];
  // 固定预留项（安检证/环保证）永远置顶、顺序固定
  const reserved = RESERVED_CERT_NAMES
    .map((n) => all.find((c: any) => c.certName === n))
    .filter((c): c is any => !!c);
  // 其余自由添加的证件按当前排序规则排
  const rest = all.filter((c: any) => !isReservedCertName(c.certName));
  rest.sort((a: any, b: any) => {
    let va = a[certSortProp.value];
    let vb = b[certSortProp.value];
    if (va == null) va = '';
    if (vb == null) vb = '';
    const cmp = String(va).localeCompare(String(vb), 'zh-CN');
    return certSortOrder.value === 'ascending' ? cmp : -cmp;
  });
  return [...reserved, ...rest];
});

function onCertSort({ prop, order }: any) {
  if (!order) return;
  certSortProp.value = prop;
  certSortOrder.value = order;
}

function openAddCert() {
  editingCertId.value = null;
  certPermFlag.value = false;
  certNameLocked.value = false;
  Object.assign(certForm, { certName: '', certNumber: '', expiryDate: '', fileId: 0, remark: '' });
  certDialog.value = true;
}

function openEditCert(row: any) {
  editingCertId.value = row.id;
  certNameLocked.value = isReservedCertName(row.certName);
  Object.assign(certForm, {
    certName: row.certName, certNumber: row.certNumber || '',
    expiryDate: row.expiryDate ? row.expiryDate.substring(0, 10) : '',
    fileId: row.fileId || 0, remark: row.remark || '',
  });
  certPermFlag.value = !row.expiryDate;
  certDialog.value = true;
}

async function saveCert() {
  await certFormRef.value?.validate();
  // 自由添加时拦截：固定预留名称不应重复添加（应去编辑对应行）
  if (!editingCertId.value && isReservedCertName(certForm.certName)
    && (detail.value?.certificates || []).some((c: any) => c.certName === certForm.certName)) {
    ElMessage.warning(`「${certForm.certName}」为固定预留项，已存在于该人员证书列表中，请直接编辑对应行`);
    return;
  }
  savingCert.value = true;
  try {
    if (editingCertId.value) {
      await personnelApi.updateCert(editingCertId.value, certForm);
      ElMessage.success('证件更新成功');
    } else {
      await personnelApi.addCert(id, certForm);
      ElMessage.success('证件添加成功');
    }
    certDialog.value = false;
    editingCertId.value = null;
    certPermFlag.value = false;
    certNameLocked.value = false;
    Object.assign(certForm, { certName: '', certNumber: '', expiryDate: '', fileId: 0, remark: '' });
    loadData();
  } finally {
    savingCert.value = false;
  }
}

async function removeCert(row: any) {
  await ElMessageBox.confirm('确认删除该证件？', '警告', { type: 'warning' });
  await personnelApi.removeCert(row.id);
  ElMessage.success('删除成功');
  loadData();
}

// ===== 文件预览 =====
const previewDialog = ref(false);
const currentFile = ref<any>(null);

function previewCert(row: any) {
  if (!row.file?.id) return; // 尚未上传文件的预留项没有内容可预览
  currentFile.value = row.file;
  previewDialog.value = true;
}

// 未上传文件的预留行给浅灰底，视觉上区分"待上传"
function certRowClass({ row }: any) {
  return row.fileId ? '' : 'cert-row-pending';
}

// ===== 工具函数 =====
function formatDate(d?: string) { return d ? dayjs(d).format('YYYY-MM-DD') : '-'; }

async function loadData() {
  loading.value = true;
  try {
    const res: any = await personnelApi.detail(id);
    detail.value = res.data;
    fillForm(res.data);
  } finally {
    loading.value = false;
  }
}

onMounted(loadData);
</script>

<style scoped lang="scss">
.cert-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cert-name-tip {
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
  margin-top: 4px;
}

:deep(.cert-row-pending) {
  background-color: #fafafa;
}
:deep(.el-table--striped .el-table__body tr.cert-row-pending.el-table__row--striped td.el-table__cell) {
  background-color: #f5f5f5;
}

.photo-upload-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.photo-tip {
  color: #909399;
  font-size: 12px;
}
.current-photo {
  width: 44px;
  height: 50px;
  border-radius: 5px;
  object-fit: cover;
  border: 1px solid #dcdfe6;
}

@media screen and (max-width: 768px) {
  .photo-upload-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
:deep(.el-form-item__label-wrap) {
  margin-right: 10px !important;
}
</style>
