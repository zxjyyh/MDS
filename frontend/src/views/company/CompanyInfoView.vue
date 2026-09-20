<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><OfficeBuilding /></el-icon>
        公司信息管理
      </div>
      <el-button v-if="authStore.isAdmin" type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>新增公司
      </el-button>
    </div>

    <div class="search-bar">
      <el-input
        v-model="query.keyword"
        placeholder="公司名 / 信用代码 / 资质证书编号"
        clearable
        style="width: 300px"
        @keyup.enter="onSearch"
        @clear="onSearch"
      />
      <el-button type="primary" @click="onSearch">查询</el-button>
      <el-button @click="resetQuery">重置</el-button>
    </div>

    <el-table
      :data="list"
      v-loading="loading"
      stripe
      border
      :row-style="{ height: '48px' }"
      :cell-style="{ padding: '4px 4px' }"
      :header-cell-style="{ padding: '8px 4px', background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
      @sort-change="onSortChange"
    >
      <el-table-column type="index" label="序号" width="55" align="center" />
      <el-table-column prop="name" label="公司名" min-width="200" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="creditCode" label="营业执照信用代码" min-width="180" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="qualCertNo" label="资质证书编号" min-width="160" sortable="custom" show-overflow-tooltip />
      <el-table-column label="资质证书到期时间" width="150" align="center" sortable="custom" prop="qualCertExpiry">
        <template #default="{ row }">
          <ExpiryBadge v-if="row.qualCertExpiry" :expiry-date="row.qualCertExpiry" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="address" label="公司地址" min-width="200" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="phone" label="公司电话" min-width="130" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="bankAccount" label="对公银行账号" min-width="180" sortable="custom" show-overflow-tooltip />
      <el-table-column prop="bankBranch" label="开户行地址" min-width="200" sortable="custom" show-overflow-tooltip />
      <el-table-column label="操作" width="150" align="center" fixed="right">
        <template #default="{ row }">
          <div style="display:flex;justify-content:center;gap:2px">
            <el-button v-if="authStore.isAdmin" size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button v-if="authStore.isAdmin" size="small" type="danger" plain @click="removeRow(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="loadData"
        @size-change="loadData"
      />
    </div>

    <!-- 新增 / 编辑公司信息 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑公司信息' : '新增公司'"
      :width="isMobile ? '94%' : '680px'"
      align-center
      destroy-on-close
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="auto" :require-asterisk-position="'right'">
        <el-form-item label="公司名" prop="name">
          <el-input v-model="form.name" placeholder="请输入公司全称" />
        </el-form-item>
        <el-row :gutter="14">
          <el-col :span="colSpan">
            <el-form-item label="营业执照信用代码">
              <el-input v-model="form.creditCode" placeholder="统一社会信用代码" />
            </el-form-item>
          </el-col>
          <el-col :span="colSpan">
            <el-form-item label="公司电话">
              <el-input v-model="form.phone" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="colSpan">
            <el-form-item label="资质证书编号">
              <el-input v-model="form.qualCertNo" />
            </el-form-item>
          </el-col>
          <el-col :span="colSpan">
            <el-form-item label="资质证书到期时间">
              <el-date-picker
                v-model="form.qualCertExpiry"
                type="date"
                value-format="YYYY-MM-DD"
                placeholder="选日期或留空"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="公司地址">
          <el-input v-model="form.address" placeholder="公司注册 / 办公地址，生成劳动合同时会作为甲方地址自动带出" />
        </el-form-item>
        <el-row :gutter="14">
          <el-col :span="colSpan">
            <el-form-item label="对公银行账号">
              <el-input v-model="form.bankAccount" />
            </el-form-item>
          </el-col>
          <el-col :span="colSpan">
            <el-form-item label="开户行地址">
              <el-input v-model="form.bankBranch" placeholder="如：中国工商银行XX支行" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { companyApi } from '../../api/company';
import { useAuthStore } from '../../stores/auth';
import ExpiryBadge from '../../components/ExpiryBadge.vue';

const authStore = useAuthStore();

const isMobile = ref(window.innerWidth <= 768);
function syncIsMobile() {
  isMobile.value = window.innerWidth <= 768;
}
window.addEventListener('resize', syncIsMobile);
// 窄屏时表单项改为单列，避免 label + 输入框被挤成两行
const colSpan = computed(() => (isMobile.value ? 24 : 12));

const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive({ page: 1, pageSize: 20, keyword: '' });

async function loadData() {
  loading.value = true;
  try {
    const res: any = await companyApi.list(query);
    list.value = res.data.list || [];
    total.value = res.data.total || 0;
  } finally {
    loading.value = false;
  }
}
function onSearch() {
  query.page = 1;
  loadData();
}
function resetQuery() {
  query.keyword = '';
  query.page = 1;
  loadData();
}

// 排序：前端按当前页排序（与设备/人员模块保持一致）
function onSortChange({ prop, order }: { prop: string | null; order: 'ascending' | 'descending' | null }) {
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

// 新增 / 编辑
const dialogVisible = ref(false);
const editingId = ref<number | null>(null);
const saving = ref(false);
const formRef = ref<FormInstance>();
const emptyForm = {
  name: '',
  creditCode: '',
  qualCertNo: '',
  qualCertExpiry: '',
  address: '',
  phone: '',
  bankAccount: '',
  bankBranch: '',
};
const form = reactive<any>({ ...emptyForm });
const rules: FormRules = {
  name: [{ required: true, message: '请输入公司名', trigger: 'blur' }],
};

function openCreate() {
  editingId.value = null;
  Object.assign(form, emptyForm);
  dialogVisible.value = true;
}
function openEdit(row: any) {
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name || '',
    creditCode: row.creditCode || '',
    qualCertNo: row.qualCertNo || '',
    qualCertExpiry: row.qualCertExpiry || '',
    address: row.address || '',
    phone: row.phone || '',
    bankAccount: row.bankAccount || '',
    bankBranch: row.bankBranch || '',
  });
  dialogVisible.value = true;
}
function resetForm() {
  editingId.value = null;
  Object.assign(form, emptyForm);
  formRef.value?.clearValidate();
}

async function save() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  saving.value = true;
  try {
    if (editingId.value) {
      await companyApi.update(editingId.value, form);
      ElMessage.success('更新成功');
    } else {
      await companyApi.create(form);
      ElMessage.success('创建成功');
    }
    dialogVisible.value = false;
    loadData();
  } catch {
    // 失败提示已由请求拦截器统一处理
  } finally {
    saving.value = false;
  }
}

async function removeRow(row: any) {
  await ElMessageBox.confirm(`确认删除公司「${row.name}」？`, '警告', { type: 'warning' });
  await companyApi.remove(row.id);
  ElMessage.success('删除成功');
  // 删掉当前页最后一条时回退一页，避免停在空白页
  if (list.value.length === 1 && query.page > 1) query.page -= 1;
  loadData();
}

onMounted(loadData);
</script>

<style scoped lang="scss">
</style>
