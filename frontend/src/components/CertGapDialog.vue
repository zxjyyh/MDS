<template>
  <el-dialog
    :model-value="modelValue"
    title="证件查漏"
    :width="dialogWidth"
    align-center
    class="cert-gap-dialog"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    @open="onOpen"
  >
    <!-- 选择区 -->
    <div class="gap-filter">
      <div class="gap-row">
        <el-select
          v-model="query.department"
          placeholder="所属检测站（不选＝全部）"
          clearable
          size="small"
          style="width: 230px"
        >
          <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
        </el-select>
        <el-select
          v-model="query.status"
          placeholder="在职状态（不选＝全部）"
          clearable
          size="small"
          style="width: 190px"
        >
          <el-option label="在职" value="active" />
          <el-option label="离职" value="resigned" />
        </el-select>
        <el-button type="primary" size="small" :loading="loading" @click="search">
          <el-icon><Search /></el-icon>查询
        </el-button>
        <el-button size="small" @click="resetQuery">重置</el-button>
      </div>

      <div class="gap-row gap-certs">
        <span class="gap-label">证书类型</span>
        <el-checkbox-group v-model="selectedCerts" size="small">
          <el-checkbox v-for="c in ALL_CERTS" :key="c" :value="c">{{ c }}</el-checkbox>
        </el-checkbox-group>
        <span class="gap-hint">全不勾选 = 查全部类型</span>
      </div>

      <div class="gap-row gap-source">
        <el-icon style="margin-right: 6px; flex-shrink: 0"><InfoFilled /></el-icon>
        <span>
          证件状态取自该人员的「证件上传」记录（按证书名称匹配）：没有该证书的上传记录即为「未上传」，
          已上传但到期日早于今天为「过期」，其余为「正常」。
        </span>
      </div>
    </div>

    <!-- 汇总条 -->
    <div v-if="searched" class="gap-summary">
      <span class="gap-sum-item">共 <b>{{ list.length }}</b> 人</span>
      <span class="gap-sum-item is-normal">正常 <b>{{ summary.normal || 0 }}</b></span>
      <span class="gap-sum-item is-missing">未上传 <b>{{ summary.missing || 0 }}</b></span>
      <span class="gap-sum-item is-expired">过期 <b>{{ summary.expired || 0 }}</b></span>
      <el-checkbox v-model="onlyProblem" size="small" class="gap-only">只看异常（未上传 / 过期）</el-checkbox>
    </div>

    <!-- 显示区 -->
    <div class="gap-result">
      <el-table
        v-if="displayList.length"
        ref="tableRef"
        :data="displayList"
        v-loading="loading"
        size="small"
        stripe
        border
        :max-height="isMobile ? 320 : 430"
        :row-style="{ height: '46px' }"
        :header-cell-style="{ background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
        @sort-change="onSortChange"
      >
        <el-table-column type="index" label="序号" width="50" align="center" />
        <el-table-column prop="name" label="姓名" width="80" align="center" sortable="custom" />
        <el-table-column prop="department" label="所属检测站" min-width="160" sortable="custom" show-overflow-tooltip>
          <template #default="{ row }">{{ row.department || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" label="在职状态" width="84" align="center" sortable="custom">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small" effect="light">
              {{ row.status === 'active' ? '在职' : '离职' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          v-for="cn in certNames"
          :key="cn"
          :prop="'cert:' + cn"
          :label="cn"
          width="104"
          align="center"
          sortable="custom"
        >
          <template #default="{ row }">
            <div class="gap-cell">
              <el-tag :type="stateType(itemOf(row, cn).state)" size="small" effect="light">
                {{ stateLabel(itemOf(row, cn).state) }}
              </el-tag>
              <span v-if="itemOf(row, cn).expiryDate" class="gap-date">{{ itemOf(row, cn).expiryDate }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty
        v-else
        :image-size="80"
        :description="loading ? '查询中…' : (searched ? '该范围内没有符合条件的人员' : '请选择条件后点击查询')"
      />
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import { personnelApi } from '../api/personnel';

// 与后端 service.CertGapCertNames 保持一致；后端也会过滤非法名，这里只是给用户看选项
const ALL_CERTS = ['安检证', '环保证', '查验证', '内审员证', '身份证', '驾驶证', '毕业证', '劳动合同'];

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>();

const isMobile = ref(window.innerWidth <= 768);
function syncIsMobile() {
  isMobile.value = window.innerWidth <= 768;
}
window.addEventListener('resize', syncIsMobile);
onUnmounted(() => window.removeEventListener('resize', syncIsMobile));

// 8 个证件类型全选时列会很多，宽屏放大到 1240px，窄屏用 94vw 兜住
const dialogWidth = computed(() => (isMobile.value ? '96%' : 'min(1240px, 94vw)'));

const departments = ref<string[]>([]);
const query = reactive({ department: '', status: '' });
const selectedCerts = ref<string[]>([...ALL_CERTS]);
const certNames = ref<string[]>([...ALL_CERTS]);
const list = ref<any[]>([]);
const summary = ref<Record<string, number>>({});
const loading = ref(false);
const searched = ref(false);
const onlyProblem = ref(false);
const tableRef = ref();

// ---- 表头点击排序（前端排当前结果集）----
// 证件列按「未上传 → 过期 → 正常」的严重度排，同状态再按到期日，这样点一下就能把有问题的顶上来
const STATE_ORDER: Record<string, number> = { missing: 0, expired: 1, normal: 2 };
const sortProp = ref('');
const sortOrder = ref<'ascending' | 'descending' | null>(null);

function onSortChange({ prop, order }: { prop: string | null; order: 'ascending' | 'descending' | null }) {
  sortProp.value = prop || '';
  sortOrder.value = order || null;
}

function compareRows(a: any, b: any) {
  const p = sortProp.value;
  if (p.startsWith('cert:')) {
    const cn = p.slice(5);
    const ia = itemOf(a, cn);
    const ib = itemOf(b, cn);
    const sa = STATE_ORDER[ia.state] ?? 9;
    const sb = STATE_ORDER[ib.state] ?? 9;
    if (sa !== sb) return sa - sb;
    return String(ia.expiryDate || '').localeCompare(String(ib.expiryDate || ''));
  }
  return String(a[p] ?? '').localeCompare(String(b[p] ?? ''), 'zh-CN');
}

const displayList = computed(() => {
  const rows = onlyProblem.value
    ? list.value.filter((r) => (r.items || []).some((i: any) => i.state !== 'normal'))
    : [...list.value];
  if (sortProp.value && sortOrder.value) {
    const dir = sortOrder.value === 'ascending' ? 1 : -1;
    rows.sort((a, b) => compareRows(a, b) * dir);
  }
  return rows;
});

function stateType(s: string) {
  return s === 'normal' ? 'success' : s === 'expired' ? 'warning' : 'danger';
}
// missing 在后端表示「证件上传里没有该证书名称」，界面文案用「未上传」
function stateLabel(s: string) {
  return s === 'normal' ? '正常' : s === 'expired' ? '过期' : '未上传';
}
// 兜底返回一个缺证项，避免表格里取不到证书时抛错
function itemOf(row: any, certName: string) {
  const hit = (row.items || []).find((i: any) => i.certName === certName);
  return hit || { certName, state: 'missing', expiryDate: null };
}

async function loadDepartments() {
  if (departments.value.length) return;
  try {
    const res: any = await personnelApi.options();
    departments.value = res.data.departments || [];
  } catch {
    /* 选项加载失败不影响查询 */
  }
}

async function search() {
  loading.value = true;
  try {
    const res: any = await personnelApi.certGap({
      department: query.department,
      status: query.status,
      certs: selectedCerts.value.join(','),
    });
    const d = res.data || {};
    list.value = d.list || [];
    certNames.value = d.certNames || [...ALL_CERTS];
    summary.value = d.summary || {};
    searched.value = true;
    // 证件类型可能变了，旧列上的排序箭头会失效 —— 重置排序状态，避免看起来"点了没反应"
    sortProp.value = '';
    sortOrder.value = null;
    tableRef.value?.clearSort();
  } finally {
    loading.value = false;
  }
}

function resetQuery() {
  query.department = '';
  query.status = '';
  selectedCerts.value = [...ALL_CERTS];
  onlyProblem.value = false;
  search();
}

// 每次打开都重新查一次，保证看到的是最新数据；筛选条件保留，便于反复核对
function onOpen() {
  loadDepartments();
  search();
}

onMounted(loadDepartments);
</script>

<style scoped lang="scss">
.gap-filter {
  background: var(--el-fill-color-light);
  border-radius: 10px;
  padding: 12px 14px;
  margin-bottom: 12px;
}
.gap-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.gap-row + .gap-row {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color);
}
.gap-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.gap-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gap-source {
  align-items: flex-start;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.gap-summary {
  display: flex;
  align-items: center;
  gap: 18px;
  flex-wrap: wrap;
  padding: 8px 14px;
  margin-bottom: 10px;
  border-radius: 10px;
  background: var(--el-color-primary-light-9);
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.gap-sum-item b {
  font-size: 15px;
  font-weight: 500;
}
.gap-sum-item.is-normal b {
  color: var(--el-color-success);
}
.gap-sum-item.is-missing b {
  color: var(--el-color-danger);
}
.gap-sum-item.is-expired b {
  color: var(--el-color-warning);
}
.gap-only {
  margin-left: auto;
}
.gap-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.gap-date {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

/* 弹窗超高时内部滚动（此处必须用 class：该版本 EP 已移除 custom-class） */
:global(.cert-gap-dialog) {
  max-height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;
}
:global(.cert-gap-dialog .el-dialog__header),
:global(.cert-gap-dialog .el-dialog__footer) {
  flex-shrink: 0;
}
:global(.cert-gap-dialog .el-dialog__body) {
  overflow: auto;
}
</style>
