<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">设备管理</span>
      <el-button v-if="authStore.isAdmin" type="primary" @click="dialogVisible = true">
        <el-icon><Plus /></el-icon>新增设备
      </el-button>
    </div>

    <div class="search-bar">
      <el-select v-model="query.department" placeholder="所属检测站" clearable style="width: 160px" @change="onDepartmentChange">
        <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
      </el-select>
      <el-select v-model="query.line" placeholder="所属检测线" clearable style="width: 160px" :disabled="!query.department">
        <el-option v-for="l in lineOptions" :key="l" :label="l" :value="l" />
      </el-select>
      <el-input v-model="query.name" placeholder="设备名称" clearable style="width: 200px" @keyup.enter="loadData" />
      <el-button type="primary" @click="loadData">查询</el-button>
      <el-button @click="resetQuery">重置</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe border
      :row-style="{ height: '48px' }"
      :cell-style="{ padding: '4px 4px' }"
      :header-cell-style="{ padding: '8px 4px', background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
      @sort-change="onSortChange"
      @row-dblclick="(row: any) => goDetail(row.id)"
    >
      <el-table-column type="index" label="序号" width="55" align="center" />
      <el-table-column prop="name" label="设备名称" min-width="140" sortable="custom" />
      <el-table-column prop="department" label="所属检测站" min-width="130" align="left" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="line" label="所属检测线" min-width="120" align="center" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="model" label="设备型号" min-width="120" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="serialNumber" label="出厂编号" min-width="130" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="manufacturer" label="厂家" min-width="120" sortable="custom" show-overflow-tooltip />
      <el-table-column label="状态" width="90" align="center" sortable="custom" prop="status">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="inspectionCertNumber" label="检定证书编号" min-width="140" sortable="custom" show-overflow-tooltip />
      <el-table-column label="检定有效期" width="115" align="center" sortable="custom" prop="inspectionExpiryDate">
        <template #default="{ row }">
          <ExpiryBadge v-if="row.inspectionExpiryDate" :expiry-date="row.inspectionExpiryDate" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="130" align="center">
        <template #default="{ row }">
          <div style="display:flex;justify-content:center;gap:2px">
            <el-button size="small" type="primary" plain @click="goDetail(row.id)">详情</el-button>
            <el-button v-if="authStore.isAdmin" size="small" type="danger" plain @click="removeRow(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :total="total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @current-change="loadData" @size-change="loadData" />
    </div>

    <!-- 新增设备对话框 -->
    <el-dialog v-model="dialogVisible" title="新增设备" :width="isMobile ? '94%' : '580px'" top="3vh" @close="resetForm">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="auto" :require-asterisk-position="'right'">
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="设备名称" prop="name"><el-input v-model="form.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="出厂编号" prop="serialNumber"><el-input v-model="form.serialNumber" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="所属检测站"><el-input v-model="form.department" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="所属检测线"><el-input v-model="form.line" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="设备型号"><el-input v-model="form.model" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="厂家"><el-input v-model="form.manufacturer" /></el-form-item></el-col>
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
            <el-date-picker v-model="form.inspectionExpiryDate" type="date" value-format="YYYY-MM-DD" style="width:200px" :disabled="permFlag" placeholder="选日期或留空" />
            <el-checkbox v-model="permFlag" style="white-space:nowrap" @change="onPermToggle">长期有效</el-checkbox>
          </div>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { equipmentApi } from '../../api/equipment';
import { useAuthStore } from '../../stores/auth';
import ExpiryBadge from '../../components/ExpiryBadge.vue';

const router = useRouter();
const authStore = useAuthStore();
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });
const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive({ page: 1, pageSize: 20, department: '', line: '', name: '' });

// 检测站 / 检测线筛选选项
const departments = ref<string[]>([]);
const linesMap = ref<Record<string, string[]>>({});
const lineOptions = computed(() => (query.department ? linesMap.value[query.department] || [] : []));
function onDepartmentChange() { query.line = ''; }
async function loadOptions() {
  try {
    const res: any = await equipmentApi.options();
    departments.value = res.data.departments || [];
    linesMap.value = res.data.lines || {};
  } catch { /* 忽略选项加载失败，不影响列表 */ }
}

// 新增对话框
const dialogVisible = ref(false);
const saving = ref(false);
const formRef = ref<FormInstance>();
const form = reactive<any>({ name: '', serialNumber: '', department: '', line: '', model: '', manufacturer: '', status: 'in_use', inspectionCertNumber: '', inspectionDate: '', inspectionExpiryDate: '', remark: '' });
const permFlag = ref(false);
function onPermToggle(v: boolean) { if (v) form.inspectionExpiryDate = ''; }
const rules: FormRules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  serialNumber: [{ required: true, message: '请输入出厂编号', trigger: 'blur' }],
};

function fmtDate(d?: string) { return d ? d.substring(0, 10) : '-'; }
function statusLabel(s: string) { return { in_use: '使用中', idle: '闲置', inspecting: '送检中', repair: '维修中', scrapped: '已报废' }[s] || s; }
function statusType(s: string): any { return { in_use: 'success', idle: 'info', inspecting: 'warning', repair: 'warning', scrapped: 'danger' }[s] || 'info'; }

// 排序
const sortKey = ref('');
const sortOrder = ref('');
function onSortChange({ prop, order }: { prop: string | null; order: 'ascending' | 'descending' | null }) {
  sortKey.value = prop || '';
  sortOrder.value = order || '';
  if (!prop || !order) { loadData(); return; }
  list.value = [...list.value].sort((a, b) => {
    const va = (a[prop] ?? '').toString();
    const vb = (b[prop] ?? '').toString();
    return order === 'ascending' ? va.localeCompare(vb, 'zh-CN') : vb.localeCompare(va, 'zh-CN');
  });
}

async function loadData() {
  loading.value = true;
  try {
    const res: any = await equipmentApi.list(query);
    list.value = res.data.list;
    total.value = res.data.total;
  } finally { loading.value = false; }
}
function resetQuery() { query.department = ''; query.line = ''; query.name = ''; query.page = 1; loadData(); }
function goDetail(id: number) { router.push(`/equipment/${id}`); }
function resetForm() { Object.assign(form, { name: '', serialNumber: '', department: '', line: '', model: '', manufacturer: '', status: 'in_use', inspectionCertNumber: '', inspectionDate: '', inspectionExpiryDate: '', remark: '' }); permFlag.value = false; }

async function save() {
  await formRef.value?.validate();
  saving.value = true;
  try {
    await equipmentApi.create(form);
    ElMessage.success('创建成功');
    dialogVisible.value = false;
    loadData();
  } finally { saving.value = false; }
}

async function removeRow(row: any) {
  await ElMessageBox.confirm(`确认删除设备「${row.name}」及其所有证书？`, '警告', { type: 'warning' });
  await equipmentApi.remove(row.id);
  ElMessage.success('删除成功');
  loadData();
}

onMounted(() => { loadData(); loadOptions(); });
</script>

<style scoped lang="scss">
</style>
