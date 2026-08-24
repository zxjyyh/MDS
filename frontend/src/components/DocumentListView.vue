<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">{{ title }}</span>
      <div class="header-buttons">
        <el-button v-if="authStore.isAdmin" @click="catDialog = true">
          <el-icon><FolderOpened /></el-icon>管理分类
        </el-button>
        <el-button v-if="authStore.isAdmin" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>上传文档
        </el-button>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :sm="5">
        <el-card shadow="never" class="category-card">
          <template #header><span>分类</span></template>
          <CategoryTree
            ref="catTreeRef"
            :system-type="systemType"
            :show-actions="authStore.isAdmin"
            @select="handleSelect"
            @add="addCategory"
            @edit="editCategory"
            @remove="removeCategory"
          />
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="19">
        <el-card shadow="never">
          <div class="search-bar">
            <el-input v-model="query.keyword" placeholder="标题/编号" clearable style="width: 200px" @keyup.enter="loadData" />
            <el-select v-model="query.expiryStatus" placeholder="到期状态" clearable style="width: 140px" @change="loadData">
              <el-option label="已过期" value="expired" />
              <el-option label="30天内到期" value="expiring_30" />
              <el-option label="60天内到期" value="expiring_60" />
              <el-option label="90天内到期" value="expiring_90" />
              <el-option label="正常" value="normal" />
            </el-select>
            <el-button type="primary" @click="loadData">查询</el-button>
            <el-button @click="resetQuery">重置</el-button>
          </div>

    <el-table :data="list" v-loading="loading" stripe border class="doc-table" @row-dblclick="previewRow">
      <!-- 公司档案：分类在最前 -->
      <el-table-column v-if="isArchive" label="分类" width="100" fixed="left">
        <template #default="{ row }">{{ row.category?.name || '-' }}</template>
      </el-table-column>
      <el-table-column type="index" label="序号" width="55" />
      <!-- 公司档案：文件名 + 按分类动态字段 -->
      <template v-if="isArchive">
        <el-table-column label="文件名" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.file?.originalName || row.title }}</template>
        </el-table-column>
        <el-table-column prop="companyName" label="公司名称" min-width="150" show-overflow-tooltip />
        <el-table-column v-if="isLicense" prop="unifiedCreditCode" label="统一信用代码" min-width="160" show-overflow-tooltip />
        <el-table-column v-if="isLicense" prop="legalPerson" label="法人" width="90" show-overflow-tooltip />
        <el-table-column v-if="isLicense" prop="address" label="地址" min-width="160" show-overflow-tooltip />
        <el-table-column v-if="isCert" prop="certNumber" label="证书编号" min-width="140" show-overflow-tooltip />
        <el-table-column v-if="isCert" label="证书有效期" width="105">
          <template #default="{ row }">{{ formatDate(row.expiryDate) }}</template>
        </el-table-column>
        <el-table-column label="证书日期" width="105">
          <template #default="{ row }">{{ formatDate(row.issueDate) }}</template>
        </el-table-column>
      </template>
      <!-- 非公司档案：通用列 -->
      <template v-else>
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
        <el-table-column prop="docNumber" label="文档编号" width="140" show-overflow-tooltip />
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column label="分类" width="120">
          <template #default="{ row }">{{ row.category?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="发布日期" width="110">
          <template #default="{ row }">{{ formatDate(row.issueDate) }}</template>
        </el-table-column>
        <el-table-column label="到期日期" width="110">
          <template #default="{ row }">{{ formatDate(row.expiryDate) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><ExpiryBadge :expiry-date="row.expiryDate" /></template>
        </el-table-column>
      </template>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button v-if="authStore.isAdmin" text size="small" @click="editDoc(row)">编辑</el-button>
          <el-button v-if="authStore.isAdmin" text size="small" type="danger" @click="removeDoc(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

          <div class="pagination-wrap">
            <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :total="total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @current-change="loadData" @size-change="loadData" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 上传/编辑文档对话框 -->
    <el-dialog v-model="docDialog" :title="editingDoc ? '编辑文档' : '上传文档'" :width="isMobile ? '94%' : '620px'">
      <el-form ref="docFormRef" :model="docForm" :rules="docRules" label-width="95px">
        <el-form-item label="分类" prop="categoryId">
          <el-tree-select
            v-model="docForm.categoryId"
            :data="categoryOptions"
            :props="{ label: 'name', children: 'children' }"
            node-key="id"
            check-strictly
            placeholder="选择分类"
            style="width: 100%"
          />
        </el-form-item>

        <!-- 公司档案：文件名 + 扩展字段 -->
        <template v-if="isArchive">
          <el-form-item label="文件名" prop="fileName">
            <el-input v-model="docForm.fileName" placeholder="上传文件后自动填入，可修改" />
          </el-form-item>
          <el-form-item label="公司名称">
            <el-input v-model="docForm.companyName" />
          </el-form-item>
          <el-form-item label="证书日期">
            <el-date-picker v-model="docForm.issueDate" type="date" value-format="YYYY-MM-DD" placeholder="手动填写证书日期" style="width: 100%" />
          </el-form-item>
          <template v-if="isLicenseForm">
            <el-row :gutter="16">
              <el-col :span="12"><el-form-item label="统一信用代码"><el-input v-model="docForm.unifiedCreditCode" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="法人"><el-input v-model="docForm.legalPerson" /></el-form-item></el-col>
            </el-row>
            <el-form-item label="地址"><el-input v-model="docForm.address" /></el-form-item>
          </template>
          <template v-if="isCertForm">
            <el-row :gutter="16">
              <el-col :span="12"><el-form-item label="证书编号"><el-input v-model="docForm.certNumber" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="证书有效期"><el-date-picker v-model="docForm.expiryDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
            </el-row>
          </template>
        </template>

        <!-- 非公司档案：标题等通用字段 -->
        <template v-else>
          <el-form-item label="标题" prop="title">
            <el-input v-model="docForm.title" />
          </el-form-item>
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="文档编号"><el-input v-model="docForm.docNumber" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="版本"><el-input v-model="docForm.version" /></el-form-item></el-col>
          </el-row>
          <el-row :gutter="16">
            <el-col :span="12"><el-form-item label="发布日期"><el-date-picker v-model="docForm.issueDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
            <el-col :span="12"><el-form-item label="到期日期"><el-date-picker v-model="docForm.expiryDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" /></el-form-item></el-col>
          </el-row>
        </template>

        <el-form-item label="文档文件" prop="fileId">
          <FileUploader @success="(f: any) => onFileUploaded(f)" />
          <el-tag v-if="docForm.fileId" type="success" closable @close="docForm.fileId = 0">已上传</el-tag>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="docForm.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="docDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveDoc">保存</el-button>
      </template>
    </el-dialog>

    <!-- 预览对话框 -->
    <el-dialog v-model="previewDialog" title="文件预览" :width="isMobile ? '94%' : '800px'" destroy-on-close>
      <FilePreview v-if="previewFile" :file-id="previewFile.id" :file-name="previewFile.originalName" :mime-type="previewFile.mimeType" />
    </el-dialog>

    <!-- 分类管理对话框 -->
    <el-dialog v-model="catDialog" title="管理分类" :width="isMobile ? '94%' : '500px'">
      <el-form :model="catForm" label-width="80px" inline>
        <el-form-item label="父分类">
          <el-tree-select v-model="catForm.parentId" :data="categoryOptions" :props="{ label: 'name', children: 'children' }" node-key="id" check-strictly clearable placeholder="顶级分类" style="width: 180px" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="catForm.name" style="width: 180px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveCategory">{{ editingCat ? '更新' : '添加' }}</el-button>
          <el-button v-if="editingCat" @click="cancelEditCat">取消</el-button>
        </el-form-item>
      </el-form>
      <CategoryTree :system-type="systemType" :show-actions="true" @add="addCategory" @edit="editCategory" @remove="removeCategory" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { categoryApi, documentApi } from '../api/document';
import { fileApi } from '../api/file';
import { useAuthStore } from '../stores/auth';
import CategoryTree from './CategoryTree.vue';
import ExpiryBadge from './ExpiryBadge.vue';
import FileUploader from './FileUploader.vue';
import FilePreview from './FilePreview.vue';
import dayjs from 'dayjs';

const props = defineProps<{ systemType: string; title: string }>();
const router = useRouter();
const authStore = useAuthStore();
const isMobile = ref(window.innerWidth <= 768);
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive<any>({ page: 1, pageSize: 20, keyword: '', expiryStatus: '', categoryId: null, systemType: props.systemType });

const categoryOptions = ref<any[]>([]);
const catTreeRef = ref();

// 是否公司档案模块
const isArchive = computed(() => props.systemType === 'company_archive');
// 当前选中分类名（决定表格列）
const selectedCategoryName = ref('');
const isLicense = computed(() => selectedCategoryName.value === '营业执照');
const isCert = computed(() => selectedCategoryName.value === '资质证书');

// 表单分类名（决定表单字段）
function findCategoryName(nodes: any[], id: any): string {
  for (const n of nodes || []) {
    if (n.id === id) return n.name;
    const r = findCategoryName(n.children, id);
    if (r) return r;
  }
  return '';
}
const docFormCategoryName = computed(() => findCategoryName(categoryOptions.value, docForm.categoryId));
const isLicenseForm = computed(() => docFormCategoryName.value === '营业执照');
const isCertForm = computed(() => docFormCategoryName.value === '资质证书');

const docDialog = ref(false);
const docFormRef = ref<FormInstance>();
const saving = ref(false);
const editingDoc = ref<any>(null);
const docForm = reactive<any>({ categoryId: null, title: '', fileName: '', companyName: '', unifiedCreditCode: '', legalPerson: '', address: '', certNumber: '', docNumber: '', version: '', issueDate: '', expiryDate: '', fileId: 0, description: '' });
const docRules = computed<FormRules>(() => ({
  categoryId: [{ required: true, message: '请选择分类' }],
  title: isArchive.value ? [] : [{ required: true, message: '请输入标题' }],
  fileId: [{ required: true, validator: (_r, v, cb) => (v ? cb() : cb(new Error('请上传文件'))) }],
}));

const catDialog = ref(false);
const catForm = reactive<any>({ parentId: null, name: '' });
const editingCat = ref<any>(null);

function formatDate(d?: string) { return d ? dayjs(d).format('YYYY-MM-DD') : '-'; }

async function loadCategories() {
  const res: any = await categoryApi.getTree(props.systemType);
  categoryOptions.value = res.data || [];
}

async function loadData() {
  loading.value = true;
  try {
    const params = { ...query, systemType: props.systemType };
    const res: any = await documentApi.list(params);
    list.value = res.data.list;
    total.value = res.data.total;
  } finally { loading.value = false; }
}

function handleSelect(node: any) {
  selectedCategoryName.value = node.name || '';
  query.categoryId = node.id;
  query.page = 1;
  loadData();
}

function resetQuery() {
  query.keyword = ''; query.expiryStatus = ''; query.categoryId = null; query.page = 1;
  selectedCategoryName.value = '';
  loadData();
}

// system_type -> 详情页路由前缀
const detailPathMap: Record<string, string> = {
  qualification: 'qualification',
  safety: 'safety',
  company_archive: 'archive/company',
  legal_standard: 'archive/standard',
};

function goDetail(id: number) {
  const base = detailPathMap[props.systemType] || props.systemType;
  router.push(`/${base}/${id}`);
}

function downloadDoc(row: any) {
  window.open(fileApi.downloadUrl(row.fileId), '_blank');
}

// 双击/预览：弹窗在线预览
const previewDialog = ref(false);
const previewFile = ref<any>(null);
function previewRow(row: any) {
  if (!row.file) {
    ElMessage.warning('该文档没有文件');
    return;
  }
  previewFile.value = row.file;
  previewDialog.value = true;
}

// 上传成功后：回填文件名（取上传文件的原始名）
function onFileUploaded(f: any) {
  docForm.fileId = f.id;
  if (isArchive.value && !docForm.fileName) {
    docForm.fileName = f.originalName || f.name || '';
  }
}

// 编辑
function editDoc(row: any) {
  editingDoc.value = row;
  Object.assign(docForm, {
    categoryId: row.categoryId,
    title: row.title || '',
    fileName: row.file?.originalName || row.title || '',
    companyName: row.companyName || '',
    unifiedCreditCode: row.unifiedCreditCode || '',
    legalPerson: row.legalPerson || '',
    address: row.address || '',
    certNumber: row.certNumber || '',
    docNumber: row.docNumber || '',
    version: row.version || '',
    issueDate: row.issueDate ? row.issueDate.substring(0, 10) : '',
    expiryDate: row.expiryDate ? row.expiryDate.substring(0, 10) : '',
    fileId: row.fileId || 0,
    description: row.description || '',
  });
  docDialog.value = true;
}

function resetDocForm() {
  editingDoc.value = null;
  Object.assign(docForm, { categoryId: null, title: '', fileName: '', companyName: '', unifiedCreditCode: '', legalPerson: '', address: '', certNumber: '', docNumber: '', version: '', issueDate: '', expiryDate: '', fileId: 0, description: '' });
}

function openCreate() {
  resetDocForm();
  docDialog.value = true;
}

async function saveDoc() {
  await docFormRef.value?.validate();
  saving.value = true;
  try {
    const payload: any = { ...docForm };
    // 公司档案：title 非必填，用文件名/公司名称兜底，避免 NOT NULL 约束
    if (isArchive.value && !payload.title) {
      payload.title = payload.fileName || payload.companyName || '未命名';
    }
    if (editingDoc.value) {
      await documentApi.update(editingDoc.value.id, payload);
      ElMessage.success('保存成功');
    } else {
      await documentApi.create({ ...payload, systemType: props.systemType });
      ElMessage.success('上传成功');
    }
    docDialog.value = false;
    resetDocForm();
    loadData();
  } finally { saving.value = false; }
}

async function removeDoc(row: any) {
  await ElMessageBox.confirm(`确认删除文档「${row.title}」？`, '警告', { type: 'warning' });
  await documentApi.remove(row.id);
  ElMessage.success('删除成功');
  loadData();
}

function addCategory(parent: any) {
  editingCat.value = null;
  catForm.parentId = parent?.id || null;
  catForm.name = '';
}

function editCategory(node: any) {
  editingCat.value = node;
  catForm.parentId = node.parentId;
  catForm.name = node.name;
}

function cancelEditCat() {
  editingCat.value = null;
  catForm.name = '';
  catForm.parentId = null;
}

async function saveCategory() {
  if (!catForm.name) { ElMessage.warning('请输入分类名称'); return; }
  if (editingCat.value) {
    await categoryApi.update(editingCat.value.id, { name: catForm.name, parentId: catForm.parentId });
    ElMessage.success('更新成功');
  } else {
    await categoryApi.create({ name: catForm.name, parentId: catForm.parentId, systemType: props.systemType });
    ElMessage.success('添加成功');
  }
  editingCat.value = null;
  catForm.name = '';
  catForm.parentId = null;
  loadCategories();
  catTreeRef.value?.loadTree();
}

async function removeCategory(node: any) {
  await ElMessageBox.confirm(`确认删除分类「${node.name}」？`, '警告', { type: 'warning' });
  await categoryApi.remove(node.id);
  ElMessage.success('删除成功');
  loadCategories();
  catTreeRef.value?.loadTree();
}

watch(() => props.systemType, () => {
  query.systemType = props.systemType;
  loadCategories();
  loadData();
});

onMounted(() => {
  loadCategories();
  loadData();
});
</script>

<style scoped lang="scss">
.header-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.category-card {
  margin-bottom: 8px;
}
@media screen and (max-width: 768px) {
  .header-buttons .el-button span {
    font-size: 13px;
  }
}
</style>
