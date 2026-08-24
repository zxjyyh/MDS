<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">{{ title }}</span>
      <div class="header-buttons">
        <el-button v-if="authStore.isAdmin" @click="catDialog = true">
          <el-icon><FolderOpened /></el-icon>管理分类
        </el-button>
        <el-button v-if="authStore.isAdmin" type="primary" @click="docDialog = true">
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

    <el-table :data="list" v-loading="loading" stripe border class="doc-table">
            <el-table-column type="index" label="#" width="50" />
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
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button text size="small" type="primary" @click="goDetail(row.id)">预览</el-button>
                <el-button text size="small" @click="downloadDoc(row)">下载</el-button>
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

    <!-- 上传文档对话框 -->
    <el-dialog v-model="docDialog" title="上传文档" :width="isMobile ? '94%' : '600px'">
      <el-form ref="docFormRef" :model="docForm" :rules="docRules" label-width="90px">
        <el-form-item label="分类" prop="categoryId">
          <el-tree-select
            v-model="docForm.categoryId"
            :data="categoryOptions"
            :props="{ label: 'name', children: 'children' }"
            check-strictly
            placeholder="选择分类"
            style="width: 100%"
          />
        </el-form-item>
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
        <el-form-item label="文档文件" prop="fileId">
          <FileUploader @success="(f: any) => (docForm.fileId = f.id)" />
          <el-tag v-if="docForm.fileId" type="success" closable @close="docForm.fileId = 0">已上传</el-tag>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="docForm.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="docDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveDoc">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分类管理对话框 -->
    <el-dialog v-model="catDialog" title="管理分类" :width="isMobile ? '94%' : '500px'">
      <el-form :model="catForm" label-width="80px" inline>
        <el-form-item label="父分类">
          <el-tree-select v-model="catForm.parentId" :data="categoryOptions" :props="{ label: 'name', children: 'children' }" check-strictly clearable placeholder="顶级分类" style="width: 180px" />
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

const docDialog = ref(false);
const docFormRef = ref<FormInstance>();
const saving = ref(false);
const docForm = reactive<any>({ categoryId: null, title: '', docNumber: '', version: '', issueDate: '', expiryDate: '', fileId: 0, description: '' });
const docRules: FormRules = {
  categoryId: [{ required: true, message: '请选择分类' }],
  title: [{ required: true, message: '请输入标题' }],
  fileId: [{ required: true, validator: (_r, v, cb) => (v ? cb() : cb(new Error('请上传文件'))) }],
};

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
  query.categoryId = node.id;
  query.page = 1;
  loadData();
}

function resetQuery() {
  query.keyword = ''; query.expiryStatus = ''; query.categoryId = null; query.page = 1;
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

async function saveDoc() {
  await docFormRef.value?.validate();
  saving.value = true;
  try {
    await documentApi.create({ ...docForm, systemType: props.systemType });
    ElMessage.success('上传成功');
    docDialog.value = false;
    Object.assign(docForm, { categoryId: null, title: '', docNumber: '', version: '', issueDate: '', expiryDate: '', fileId: 0, description: '' });
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
