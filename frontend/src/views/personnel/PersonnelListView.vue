<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">人员管理</span>
    </div>

    <div class="search-bar">
      <el-input v-model="query.keyword" placeholder="姓名" clearable style="width: 200px" @keyup.enter="loadData" />
      <el-select v-model="query.department" placeholder="所属检测站" clearable style="width: 160px">
        <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
      </el-select>
      <el-select v-model="query.status" placeholder="在职状态" clearable style="width: 120px">
        <el-option label="在职" value="active" />
        <el-option label="离职" value="resigned" />
      </el-select>
      <el-button type="primary" @click="loadData">查询</el-button>
      <el-button @click="resetQuery">重置</el-button>
      <el-button type="warning" @click="certGapDialog = true">
        <el-icon><Search /></el-icon>证件查漏
      </el-button>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>新增人员
      </el-button>
      <el-button type="danger" @click="colDialog = true">
        <el-icon><Setting /></el-icon>展示设置
      </el-button>
    </div>

    <el-table
      :data="list"
      v-loading="loading"
      stripe
      border
      :row-style="{ height: '56px' }"
      :cell-style="{ padding: '5px 4px' }"
      :header-cell-style="{ padding: '10px 4px', background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
      class="personnel-table"
      @sort-change="onSortChange"
      @row-dblclick="(row: any) => goDetail(row.id)"
    >
      <el-table-column type="index" label="序号" min-width="50" align="center" />
      <!-- 员工头像：矩形，点击放大预览 -->
      <el-table-column v-if="isVisible('photo')" label="头像" min-width="60" align="center">
        <template #default="{ row }">
          <div
            class="photo-box"
            :class="{ 'has-photo': row.photoFile }"
            @click="row.photoFile && openPreview(row.photoFile.id, '员工照片')"
          >
            <img v-if="row.photoFile" :src="fileApi.previewUrl(row.photoFile.id)" class="photo-img" alt="照片" />
            <el-icon v-else :size="20" class="photo-placeholder"><UserFilled /></el-icon>
          </div>
        </template>
      </el-table-column>
      <!-- 姓名：可排序 -->
      <el-table-column v-if="isVisible('name')" prop="name" label="姓名" min-width="70" align="center" sortable="custom" />
      <!-- 电话：可排序 -->
      <el-table-column v-if="isVisible('phone')" prop="phone" label="电话" min-width="120" align="center" show-overflow-tooltip sortable="custom" />
      <!-- 身份证号：点击预览身份证照片，可排序 -->
      <el-table-column v-if="isVisible('idCard')" prop="idCard" label="身份证号" min-width="180" align="center" sortable="custom">
        <template #default="{ row }">
          <span
            :class="{ 'clickable-text': getCert(row, 'id_card') }"
            @click="previewCert(row, 'id_card')"
          >{{ row.idCard || '-' }}</span>
        </template>
      </el-table-column>
      <!-- 学历：可排序 -->
      <el-table-column v-if="isVisible('education')" prop="education" label="学历" min-width="100" align="center" sortable="custom">
        <template #default="{ row }">
          {{ row.education || '-' }}
        </template>
      </el-table-column>
      <!-- 驾驶证：可排序 -->
      <el-table-column v-if="isVisible('driverLicense')" prop="driverLicense" label="驾驶证" min-width="90" align="center" sortable="custom">
        <template #default="{ row }">
          {{ row.driverLicense || '-' }}
        </template>
      </el-table-column>
      <!-- 所属检测站：可排序 -->
      <el-table-column v-if="isVisible('department')" prop="department" label="所属检测站" min-width="120" align="left" show-overflow-tooltip sortable="custom" />
      <!-- 在职状态：无反馈 -->
      <el-table-column v-if="isVisible('status')" label="在职状态" min-width="85" align="center" prop="status" sortable="custom">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small" effect="light">
            {{ row.status === 'active' ? '在职' : '离职' }}
          </el-tag>
        </template>
      </el-table-column>
      <!-- 安检有效期 -->
      <el-table-column v-if="isVisible('securityCertDate')" label="安检有效期" width="105" align="center" sortable="custom" prop="securityCertDate">
        <template #default="{ row }">
          <ExpiryBadge v-if="row.securityCertDate" :expiry-date="row.securityCertDate" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <!-- 环保有效期 -->
      <el-table-column v-if="isVisible('envCertDate')" label="环保有效期" width="105" align="center" sortable="custom" prop="envCertDate">
        <template #default="{ row }">
          <ExpiryBadge v-if="row.envCertDate" :expiry-date="row.envCertDate" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <!-- 查验有效期 -->
      <el-table-column v-if="isVisible('inspectionDate')" label="查验有效期" width="105" align="center" sortable="custom" prop="inspectionDate">
        <template #default="{ row }">
          <ExpiryBadge v-if="row.inspectionDate" :expiry-date="row.inspectionDate" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <!-- 安检岗位（默认隐藏，需在「展示设置」里勾选） -->
      <el-table-column v-if="isVisible('securityPositions')" label="安检岗位" min-width="150" align="left" prop="securityPositions" sortable="custom" show-overflow-tooltip>
        <template #default="{ row }">{{ row.securityPositions || '-' }}</template>
      </el-table-column>
      <!-- 环保岗位（默认隐藏，需在「展示设置」里勾选） -->
      <el-table-column v-if="isVisible('envPositions')" label="环保岗位" min-width="150" align="left" prop="envPositions" sortable="custom" show-overflow-tooltip>
        <template #default="{ row }">{{ row.envPositions || '-' }}</template>
      </el-table-column>
      <!-- 操作 -->
      <el-table-column label="操作" width="140" align="center">
        <template #default="{ row }">
          <div style="display:flex;justify-content:center;gap:2px">
            <el-button size="small" type="primary" plain @click="goDetail(row.id)">详情</el-button>
            <el-button size="small" type="danger" plain @click="removeRow(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        @current-change="loadData"
        @size-change="loadData"
      />
    </div>

    <!-- 图片放大预览遮罩 -->
    <Teleport to="body">
      <Transition name="preview-fade">
        <div v-if="previewVisible" class="image-overlay" @click.self="previewVisible = false">
          <img v-if="previewUrl" :src="previewUrl" :alt="previewTitle" class="overlay-img" />
        </div>
      </Transition>
    </Teleport>

    <!-- 新增人员对话框 -->
    <el-dialog v-model="dialogVisible" title="新增人员" :width="isMobile ? '94%' : '640px'" align-center @close="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="auto" :require-asterisk-position="'right'">
        <el-row :gutter="16">
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
        <el-form-item label="住址">
          <el-input v-model="form.address" style="width:100%;max-width:360px" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="身份证号">
              <el-input v-model="form.idCard" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="所属检测站">
              <el-input v-model="form.department" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="性别" prop="gender">
              <el-select v-model="form.gender" placeholder="请选择" style="width: 100%">
                <el-option label="男" value="male" />
                <el-option label="女" value="female" />
                <el-option label="未知" value="unknown" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="职称">
              <el-input v-model="form.position" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
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
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="员工照片">
              <FileUploader accept="image/*" @success="(f: any) => (form.photoFileId = f.id)" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" style="width:100%;max-width:360px" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="学历">
              <el-select v-model="form.education" style="width: 100%" clearable>
                <el-option v-for="e in EDUCATION_OPTIONS" :key="e" :label="e" :value="e" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="毕业学校">
              <el-input v-model="form.school" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="专业">
          <el-input v-model="form.major" style="width:100%;max-width:260px" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="毕业时间">
              <el-date-picker v-model="form.graduationDate" type="date" value-format="YYYY-MM-DD" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="身份证到期">
              <el-date-picker v-model="form.idCardExpiry" type="date" value-format="YYYY-MM-DD" style="width:100%" placeholder="到期时间" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="驾驶证">
              <el-input v-model="form.driverLicense" placeholder="证号" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="驾照到期">
              <el-date-picker v-model="form.driverLicenseExpiry" type="date" value-format="YYYY-MM-DD" style="width:100%" placeholder="到期时间" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="安检证">
          <div style="width:100%">
            <el-row :gutter="10">
              <el-col :span="10"><el-input v-model="form.securityCertNumber" placeholder="编号" /></el-col>
              <el-col :span="14"><el-date-picker v-model="form.securityCertDate" type="date" value-format="YYYY-MM-DD" style="width:100%" placeholder="到期时间" /></el-col>
            </el-row>
            <el-select
              v-model="form.securityPositions"
              multiple
              clearable
              collapse-tags
              collapse-tags-tooltip
              placeholder="安检岗位（非必选，可多选）"
              style="width:100%;margin-top:6px"
            >
              <el-option v-for="p in SECURITY_POSITION_OPTIONS" :key="p" :label="p" :value="p" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="环保证">
          <div style="width:100%">
            <el-row :gutter="10">
              <el-col :span="10"><el-input v-model="form.envCertNumber" placeholder="编号" /></el-col>
              <el-col :span="14"><el-date-picker v-model="form.envCertDate" type="date" value-format="YYYY-MM-DD" style="width:100%" placeholder="到期时间" /></el-col>
            </el-row>
            <el-select
              v-model="form.envPositions"
              multiple
              clearable
              collapse-tags
              collapse-tags-tooltip
              placeholder="环保岗位（非必选，可多选）"
              style="width:100%;margin-top:6px"
            >
              <el-option v-for="p in ENV_POSITION_OPTIONS" :key="p" :label="p" :value="p" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="查验证">
          <el-row :gutter="10" style="width:100%">
            <el-col :span="10"><el-input v-model="form.inspectionCertNumber" placeholder="编号" /></el-col>
            <el-col :span="14"><el-date-picker v-model="form.inspectionDate" type="date" value-format="YYYY-MM-DD" style="width:100%" placeholder="到期时间" /></el-col>
          </el-row>
        </el-form-item>
        <div class="form-tip">
          提示：身份证照片请在创建后进入「详情」页面上传。
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- 证件查漏：按检测站/在职状态 + 证书类型列出人员的证件状态 -->
    <CertGapDialog v-model="certGapDialog" />

    <!-- 展示设置：勾选列表要展示的字段（按当前账号保存，下次进入自动沿用） -->
    <ColumnSettingDialog
      v-model="colDialog"
      v-model:visible="visibleColumns"
      :columns="allColumns"
      @reset="resetColumns"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { personnelApi } from '../../api/personnel';
import { fileApi } from '../../api/file';
import { useAuthStore } from '../../stores/auth';
import FileUploader from '../../components/FileUploader.vue';
import ExpiryBadge from '../../components/ExpiryBadge.vue';
import ColumnSettingDialog from '../../components/ColumnSettingDialog.vue';
import CertGapDialog from '../../components/CertGapDialog.vue';
import { useColumnSetting, type ColumnDef } from '../../composables/useColumnSetting';
import {
  EDUCATION_OPTIONS,
  SECURITY_POSITION_OPTIONS,
  ENV_POSITION_OPTIONS,
  joinMultiSelect,
} from '../../utils/constants';

const router = useRouter();
const authStore = useAuthStore();

// 展示设置：可勾选的列表字段（序号与操作列始终显示，不在此列）
const allColumns: ColumnDef[] = [
  { prop: 'photo', label: '头像' },
  { prop: 'name', label: '姓名' },
  { prop: 'phone', label: '电话' },
  { prop: 'idCard', label: '身份证号' },
  { prop: 'education', label: '学历' },
  { prop: 'driverLicense', label: '驾驶证' },
  { prop: 'department', label: '所属检测站' },
  { prop: 'status', label: '在职状态' },
  { prop: 'securityCertDate', label: '安检有效期' },
  { prop: 'envCertDate', label: '环保有效期' },
  { prop: 'inspectionDate', label: '查验有效期' },
  // 岗位类字段默认不显示：它们只在"排班/报名时看谁适合干什么"的场景才有用，
  // 日常看人员列表时只会占宽度。需要的人自己去「展示设置」勾上。
  { prop: 'securityPositions', label: '安检岗位', defaultHidden: true },
  { prop: 'envPositions', label: '环保岗位', defaultHidden: true },
];
const colDialog = ref(false);
// 证件查漏弹窗
const certGapDialog = ref(false);
const { visibleColumns, isVisible, resetColumns } = useColumnSetting('personnel', allColumns);
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
// 默认一页 100 条：本机构人员规模在几十人量级，20 条会导致频繁翻页（用户 2026-09-22 要求）
const query = reactive({ page: 1, pageSize: 100, keyword: '', department: '', status: '' });

// 检测站筛选选项
const departments = ref<string[]>([]);
async function loadOptions() {
  try {
    const res: any = await personnelApi.options();
    departments.value = res.data.departments || [];
  } catch { /* 忽略选项加载失败，不影响列表 */ }
}

// ===== 图片预览 =====
const previewVisible = ref(false);
const previewUrl = ref('');
const previewTitle = ref('');

function openPreview(fileId: number, title: string) {
  previewUrl.value = fileApi.previewUrl(fileId);
  previewTitle.value = title;
  previewVisible.value = true;
}

// 通过证件名称关键字匹配证件（不再依赖 certType 字段）
const CERT_KEYWORDS: Record<string, string[]> = {
  id_card: ['身份证'],
  diploma: ['学历', '毕业', '学位'],
  driver_license: ['驾驶证'],
};

function getCert(row: any, certKey: string) {
  const keywords = CERT_KEYWORDS[certKey] || [];
  return row.certificates?.find((c: any) =>
    keywords.some((kw) => (c.certName || '').includes(kw))
  );
}
// 点击身份证号/学历/驾驶证 → 预览对应证书照片
function previewCert(row: any, certKey: string) {
  const cert = getCert(row, certKey);
  if (!cert?.file) return; // 没有证件文件，不做反馈
  const labels: Record<string, string> = {
    id_card: '身份证照片',
    diploma: '学历照片',
    driver_license: '驾驶证照片',
  };
  openPreview(cert.file.id, `${row.name} - ${labels[certKey] || '证件照片'}`);
}

// ===== 表单 =====
const dialogVisible = ref(false);
const saving = ref(false);
const formRef = ref<FormInstance>();
const form = reactive<any>({
  // gender 刻意留空而不是默认 'unknown'：它现在是必选项，
  // 若给个"合法默认值"，校验会静默通过、使用者永远注意不到自己没选（用户 2026-09-24 要求）
  name: '', gender: '', idCard: '', phone: '', department: '', address: '', position: '',
  entryDate: '', education: '', school: '', major: '', driverLicense: '',
  driverLicenseExpiry: '', idCardExpiry: '', graduationDate: '',
  securityCertDate: '', envCertDate: '', inspectionDate: '',
  securityCertNumber: '', envCertNumber: '', inspectionCertNumber: '',
  // 岗位是多选，表单里保持数组；提交时用 joinMultiSelect 转成逗号分隔字符串
  securityPositions: [], envPositions: [],
  status: 'active', remark: '', photoFileId: 0,
});
const rules: FormRules = {
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  gender: [{ required: true, message: '请选择性别', trigger: 'change' }],
};

function openCreate() {
  resetForm();
  dialogVisible.value = true;
}

async function loadData() {
  loading.value = true;
  try {
    const res: any = await personnelApi.list(query);
    list.value = res.data.list;
    total.value = res.data.total;
  } finally {
    loading.value = false;
  }
}

// 表头排序：本地排序，不调后端
const sortKey = ref('');
const sortOrder = ref('');

function onSortChange({ prop, order }: { prop: string | null; order: 'ascending' | 'descending' | null }) {
  sortKey.value = prop || '';
  sortOrder.value = order || '';
  if (!prop || !order) {
    loadData();
    return;
  }
  list.value = [...list.value].sort((a, b) => {
    const va = (a[prop] ?? '').toString();
    const vb = (b[prop] ?? '').toString();
    return order === 'ascending' ? va.localeCompare(vb, 'zh-CN') : vb.localeCompare(va, 'zh-CN');
  });
}

function resetQuery() {
  query.keyword = ''; query.department = ''; query.status = ''; query.page = 1;
  loadData();
}

function goDetail(id: number) {
  router.push(`/personnel/${id}`);
}

function resetForm() {
  Object.assign(form, {
    id: undefined, name: '', gender: '', idCard: '', phone: '',
    department: '', address: '', position: '', entryDate: '', education: '', school: '', major: '', driverLicense: '',
    driverLicenseExpiry: '', idCardExpiry: '', graduationDate: '',
    securityCertDate: '', envCertDate: '', inspectionDate: '',
    securityCertNumber: '', envCertNumber: '', inspectionCertNumber: '',
    securityPositions: [], envPositions: [],
    status: 'active', remark: '', photoFileId: 0,
  });
}

async function save() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    const payload: any = {
      ...form,
      securityPositions: joinMultiSelect(form.securityPositions),
      envPositions: joinMultiSelect(form.envPositions),
    };
    if (!payload.photoFileId) delete payload.photoFileId;
    await personnelApi.create(payload);
    ElMessage.success('创建成功');
    dialogVisible.value = false;
    loadData();
  } finally {
    saving.value = false;
  }
}

async function removeRow(row: any) {
  await ElMessageBox.confirm(`确认删除人员「${row.name}」及其所有证件？`, '警告', { type: 'warning' });
  await personnelApi.remove(row.id);
  ElMessage.success('删除成功');
  loadData();
}

onMounted(() => { loadData(); loadOptions(); });
</script>

<style scoped lang="scss">
/* 矩形头像照片框 */
.photo-box {
  width: 44px;
  height: 50px;
  border-radius: 5px;
  overflow: hidden;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  transition: all 0.2s;

  .photo-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .photo-placeholder {
    color: var(--el-text-color-placeholder);
  }

  &.has-photo {
    cursor: pointer;
    border-color: var(--el-color-primary-light-5);
    &:hover {
      box-shadow: 0 0 0 2px var(--el-color-primary);
      transform: scale(1.03);
    }
  }
}

/* 表格列宽由浏览器根据内容 + min-width 自动决定 */
:deep(.personnel-table) {
  width: 100%;
}
/* body 单元格不换行，配合 min-width 保持列宽 */
:deep(.personnel-table td) {
  white-space: nowrap;
}

.clickable-text {
  color: var(--el-color-primary);
  cursor: pointer;
  text-decoration: underline;
  &:hover {
    opacity: 0.8;
  }
}

/* 图片放大预览遮罩 */
.image-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  cursor: pointer;
}
.overlay-img {
  max-width: 520px;
  max-height: 70vh;
  object-fit: contain;
  border-radius: 6px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  animation: preview-zoom-in 0.22s ease-out;
}
@keyframes preview-zoom-in {
  from { transform: scale(0.9); opacity: 0; }
  to   { transform: scale(1);   opacity: 1; }
}

/* Vue Transition 类 */
.preview-fade-enter-active { transition: opacity 0.2s ease; }
.preview-fade-leave-active { transition: opacity 0.18s ease; }
.preview-fade-enter-from,
.preview-fade-leave-to       { opacity: 0; }

.form-tip {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  margin-top: 8px;
}

</style>
