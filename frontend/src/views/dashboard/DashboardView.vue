<template>
  <div class="dashboard page-container">
    <div class="page-header">
      <span class="page-title">仪表盘</span>
      <el-button v-if="authStore.isAdmin" type="primary" plain @click="testRun" :loading="testing">
        手动触发到期检查
      </el-button>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :xs="24" :sm="12" :md="6" v-for="card in cards" :key="card.key">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card-header">
            <el-icon :size="20" :color="card.color"><component :is="card.icon" /></el-icon>
            <span>{{ card.title }}</span>
          </div>
          <div class="stat-items">
            <div
              v-for="item in card.items"
              :key="item.label"
              class="stat-item"
              @click="goExpiring(card.module, item.days)"
            >
              <span class="stat-label" :style="{ color: item.color }">{{ item.label }}</span>
              <span class="stat-value">{{ item.value }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 到期明细 -->
    <el-card shadow="never" class="detail-card">
      <template #header>
        <div class="detail-header">
          <span>到期预警明细</span>
          <el-radio-group v-model="currentTab" size="small" @change="loadExpiring">
            <el-radio-button label="30">30天内</el-radio-button>
            <el-radio-button label="60">60天内</el-radio-button>
            <el-radio-button label="90">90天内</el-radio-button>
            <el-radio-button label="0">已过期</el-radio-button>
          </el-radio-group>
        </div>
      </template>
      <el-table :data="expiringList" v-loading="loading" stripe>
        <el-table-column label="模块" width="100" prop="module" sortable>
          <template #default="{ row }">
            <el-tag size="small">{{ moduleLabel(row.module) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="relatedName" label="人员/设备" min-width="100" sortable show-overflow-tooltip />
        <el-table-column prop="department" label="所属检测站" min-width="110" sortable show-overflow-tooltip />
        <el-table-column prop="targetName" label="证书类型" min-width="100" sortable show-overflow-tooltip />
        <el-table-column prop="expiryDate" label="到期时间" width="115" sortable>
          <template #default="{ row }">
            {{ row.expiryDate ? formatDate(row.expiryDate) : '长期有效' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template #default="{ row }">
            <el-button text size="small" type="warning" @click="doIgnore(row)">忽略</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadExpiring"
        />
      </div>
    </el-card>

    <!-- 已忽略预警 -->
    <el-card shadow="never" class="detail-card" style="margin-top: 16px">
      <template #header>
        <div class="detail-header" style="cursor: pointer" @click="ignoredExpanded = !ignoredExpanded">
          <span>
            <el-icon style="margin-right: 4px; transition: transform 0.3s" :style="{ transform: ignoredExpanded ? 'rotate(90deg)' : 'rotate(0)' }"><ArrowRight /></el-icon>
            已忽略预警 ({{ ignoredList.length }})
          </span>
        </div>
      </template>
      <template v-if="ignoredExpanded">
        <el-table :data="pagedIgnoredList" v-if="ignoredList.length" stripe>
          <el-table-column label="模块" width="100">
            <template #default="{ row }">
              <el-tag size="small">{{ moduleLabel(row.module) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="relatedName" label="人员/设备" min-width="100" show-overflow-tooltip />
          <el-table-column prop="targetName" label="证书类型" min-width="100" show-overflow-tooltip />
          <el-table-column label="到期时间" width="115">
            <template #default="{ row }">{{ row.expiryDate ? formatDate(row.expiryDate) : '-' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center">
            <template #default="{ row }">
              <el-button text size="small" type="primary" @click="doUnignore(row)">取消忽略</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-wrap" v-if="ignoredList.length">
          <el-pagination
            v-model:current-page="ignoredPage"
            v-model:page-size="ignoredPageSize"
            :total="ignoredList.length"
            layout="total, prev, pager, next"
            :page-sizes="[5, 10, 20]"
          />
        </div>
        <div v-if="!ignoredList.length" style="color: #909399; text-align: center; padding: 20px">暂无已忽略的预警</div>
      </template>
    </el-card>

    <!-- 文件预览对话框 -->
    <el-dialog v-model="previewDialog" title="文件预览" :width="dialogWidth" destroy-on-close>
      <FilePreview v-if="previewFileId" :file-id="previewFileId" :file-name="previewFileName" :mime-type="previewMime" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { dashboardApi, reminderApi } from '../../api/dashboard';
import { useAuthStore } from '../../stores/auth';
import ExpiryBadge from '../../components/ExpiryBadge.vue';
import FilePreview from '../../components/FilePreview.vue';
import dayjs from 'dayjs';

const router = useRouter();
const authStore = useAuthStore();

const isMobile = ref(window.innerWidth <= 768);
const dialogWidth = computed(() => (isMobile.value ? '94%' : '800px'));
window.addEventListener('resize', () => { isMobile.value = window.innerWidth <= 768; });

const stats = ref<any>({});
const loading = ref(false);
const testing = ref(false);

const cards = computed(() => [
  {
    key: 'personnelIdcard',
    title: '人员证件',
    icon: 'User',
    color: '#409eff',
    module: 'personnel_idcard',
    items: [
      { label: '已过期', value: stats.value.personnelIdcard?.expired || 0, color: '#f56c6c', days: 0 },
      { label: '30天内', value: stats.value.personnelIdcard?.expiring30 || 0, color: '#e6a23c', days: 30 },
      { label: '60天内', value: stats.value.personnelIdcard?.expiring60 || 0, color: '#f0c040', days: 60 },
      { label: '90天内', value: stats.value.personnelIdcard?.expiring90 || 0, color: '#409eff', days: 90 },
      { label: '总数', value: stats.value.personnelIdcard?.total || 0, color: '#67c23a', days: -1 },
    ],
  },
  {
    key: 'personnelLicense',
    title: '从业证书',
    icon: 'Stamp',
    color: '#67c23a',
    module: 'personnel_license',
    items: [
      { label: '已过期', value: stats.value.personnelLicense?.expired || 0, color: '#f56c6c', days: 0 },
      { label: '30天内', value: stats.value.personnelLicense?.expiring30 || 0, color: '#e6a23c', days: 30 },
      { label: '60天内', value: stats.value.personnelLicense?.expiring60 || 0, color: '#f0c040', days: 60 },
      { label: '90天内', value: stats.value.personnelLicense?.expiring90 || 0, color: '#409eff', days: 90 },
      { label: '总数', value: stats.value.personnelLicense?.total || 0, color: '#67c23a', days: -1 },
    ],
  },
  {
    key: 'equipmentCert',
    title: '设备证书',
    icon: 'Setting',
    color: '#e6a23c',
    module: 'equipment_inspection',
    items: [
      { label: '已过期', value: stats.value.equipmentCert?.expired || 0, color: '#f56c6c', days: 0 },
      { label: '30天内', value: stats.value.equipmentCert?.expiring30 || 0, color: '#e6a23c', days: 30 },
      { label: '60天内', value: stats.value.equipmentCert?.expiring60 || 0, color: '#f0c040', days: 60 },
      { label: '90天内', value: stats.value.equipmentCert?.expiring90 || 0, color: '#409eff', days: 90 },
      { label: '总数', value: stats.value.equipmentCert?.total || 0, color: '#67c23a', days: -1 },
    ],
  },
  {
    key: 'qualificationDoc',
    title: '资质体系文档',
    icon: 'Folder',
    color: '#e6a23c',
    module: 'qualification_doc',
    items: [
      { label: '已过期', value: stats.value.qualificationDoc?.expired || 0, color: '#f56c6c', days: 0 },
      { label: '30天内', value: stats.value.qualificationDoc?.expiring30 || 0, color: '#e6a23c', days: 30 },
      { label: '60天内', value: stats.value.qualificationDoc?.expiring60 || 0, color: '#f0c040', days: 60 },
      { label: '90天内', value: stats.value.qualificationDoc?.expiring90 || 0, color: '#409eff', days: 90 },
      { label: '总数', value: stats.value.qualificationDoc?.total || 0, color: '#67c23a', days: -1 },
    ],
  },
  {
    key: 'safetyDoc',
    title: '安全生产文档',
    icon: 'Document',
    color: '#f56c6c',
    module: 'safety_doc',
    items: [
      { label: '已过期', value: stats.value.safetyDoc?.expired || 0, color: '#f56c6c', days: 0 },
      { label: '30天内', value: stats.value.safetyDoc?.expiring30 || 0, color: '#e6a23c', days: 30 },
      { label: '60天内', value: stats.value.safetyDoc?.expiring60 || 0, color: '#f0c040', days: 60 },
      { label: '90天内', value: stats.value.safetyDoc?.expiring90 || 0, color: '#409eff', days: 90 },
      { label: '总数', value: stats.value.safetyDoc?.total || 0, color: '#67c23a', days: -1 },
    ],
  },
  {
    key: 'ignored',
    title: '已忽略预警',
    icon: 'Hide',
    color: '#909399',
    module: '',
    items: [
      { label: '总数', value: stats.value.ignored || 0, color: '#909399', days: -2 },
    ],
  },
]);

// 到期明细
const currentTab = ref('30');
const expiringList = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(10);
const total = ref(0);
const currentModule = ref('personnel_cert');

const previewDialog = ref(false);
const previewFileId = ref(0);
const previewFileName = ref('');
const previewMime = ref('');

// 已忽略预警
const ignoredList = ref<any[]>([]);
const ignoredExpanded = ref(false);
const ignoredPage = ref(1);
const ignoredPageSize = ref(5);
const pagedIgnoredList = computed(() => {
  const start = (ignoredPage.value - 1) * ignoredPageSize.value;
  return ignoredList.value.slice(start, start + ignoredPageSize.value);
});

async function loadIgnored() {
  const res: any = await dashboardApi.listIgnored();
  ignoredList.value = res.data || [];
}

async function doIgnore(row: any) {
  await dashboardApi.ignore({
    module: row.module,
    targetName: row.targetName,
    relatedName: row.relatedName || '',
    expiryDate: row.expiryDate || '',
  });
  ElMessage.success('已忽略该预警');
  loadStats();
  loadExpiring();
  loadIgnored();
}

async function doUnignore(row: any) {
  await dashboardApi.unignore(row.id);
  ElMessage.success('已取消忽略');
  loadStats();
  loadExpiring();
  loadIgnored();
}

function goIgnored() {
  // 点击已忽略卡片时滚动到已忽略预警区域
  document.querySelector('.detail-card:last-of-type')?.scrollIntoView({ behavior: 'smooth' });
}

function moduleLabel(m: string) {
  const map: Record<string, string> = {
    personnel_idcard: '人员证件',
    personnel_license: '从业证书',
    equipment_inspection: '设备证书',
    qualification_doc: '资质文档',
    safety_doc: '安全文档',
  };
  return map[m] || m;
}

function formatDate(d: string) {
  return d ? dayjs(d).format('YYYY-MM-DD') : '-';
}

function goExpiring(module: string, days: number) {
  if (days === -2) {
    goIgnored();
    return;
  }
  if (days === -1) {
    // 跳转对应模块列表
    const routes: Record<string, string> = {
      personnel_cert: '/personnel',
      equipment_cert: '/equipment',
      qualification_doc: '/qualification',
      safety_doc: '/safety',
    };
    router.push(routes[module] || '/dashboard');
    return;
  }
  currentModule.value = module;
  currentTab.value = String(days);
  page.value = 1;
  loadExpiring();
}

async function loadStats() {
  const res: any = await dashboardApi.stats();
  stats.value = res.data;
}

async function loadExpiring() {
  loading.value = true;
  try {
    const res: any = await dashboardApi.expiring({
      module: currentModule.value,
      days: parseInt(currentTab.value),
      page: page.value,
      pageSize: pageSize.value,
    });
    expiringList.value = res.data.list;
    total.value = res.data.total;
  } finally {
    loading.value = false;
  }
}

function previewFile(fileId: number) {
  previewFileId.value = fileId;
  previewDialog.value = true;
}

async function testRun() {
  testing.value = true;
  try {
    await reminderApi.testRun();
    ElMessage.success('已触发到期检查');
  } finally {
    testing.value = false;
  }
}

onMounted(() => {
  loadStats();
  loadExpiring();
  loadIgnored();
});
</script>

<style scoped lang="scss">
.stat-row {
  margin-bottom: 16px;
}
.stat-card {
  margin-bottom: 8px;
  .stat-card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    margin-bottom: 12px;
  }
  .stat-items {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .stat-item {
    display: flex;
    justify-content: space-between;
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 4px;
    &:hover {
      background: #f5f7fa;
    }
    .stat-label {
      font-size: 13px;
    }
    .stat-value {
      font-weight: 600;
    }
  }
}
.detail-card {
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }
}
@media screen and (max-width: 768px) {
  .detail-card .detail-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
