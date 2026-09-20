<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><Tickets /></el-icon>
        操作日志
      </div>
      <el-button type="danger" plain @click="clearAll">
        <el-icon><Delete /></el-icon>清空日志
      </el-button>
    </div>

    <div class="search-bar">
      <el-input
        v-model="query.keyword"
        placeholder="操作人 / 动作 / 请求路径"
        clearable
        style="width: 260px"
        @keyup.enter="onSearch"
        @clear="onSearch"
      />
      <el-select v-model="query.role" placeholder="角色" clearable style="width: 130px" @change="onSearch">
        <el-option label="管理员" value="admin" />
        <el-option label="普通用户" value="user" />
      </el-select>
      <el-button type="primary" @click="onSearch">查询</el-button>
      <el-button @click="resetQuery">重置</el-button>
    </div>

    <div class="log-tip">
      只记录有审计价值的操作：新增 / 修改 / 删除等写操作、登录退出与改密、文件上传下载导出。
      纯浏览列表与详情不会记录，列表里的头像缩略图预览也不记 ——
      否则光打开一次人员管理就会刷出几十条「预览文件」，把日志淹掉。
    </div>

    <el-table
      :data="list"
      v-loading="loading"
      stripe
      border
      :row-style="{ height: '44px' }"
      :cell-style="{ padding: '4px 4px' }"
      :header-cell-style="{ padding: '8px 4px', background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
      @sort-change="onSortChange"
    >
      <el-table-column type="index" label="序号" width="55" align="center" />
      <el-table-column label="时间" width="165" align="center" sortable="custom" prop="createdAt">
        <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作人" min-width="120" sortable="custom" prop="username">
        <template #default="{ row }">
          <span :class="{ 'deleted-user': !row.username }">{{ row.username || '（用户已删除）' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="100" align="center" sortable="custom" prop="role">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
            {{ row.role === 'admin' ? '管理员' : '普通用户' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="action" label="操作" min-width="150" sortable="custom" show-overflow-tooltip />
      <el-table-column label="请求" min-width="250" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="req-path">{{ row.method }} {{ row.path }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作对象" width="150" align="center" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.targetType">{{ row.targetType }}<template v-if="row.targetId"> #{{ row.targetId }}</template></span>
          <span v-else>-</span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import dayjs from 'dayjs';
import { operationLogApi } from '../../api/operationLog';

const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive({ page: 1, pageSize: 20, keyword: '', role: '' });

function fmtTime(t?: string) {
  if (!t) return '-';
  const d = dayjs(t);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : t;
}

async function loadData() {
  loading.value = true;
  try {
    const res: any = await operationLogApi.list(query);
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
  query.role = '';
  query.page = 1;
  loadData();
}

// 排序：前端按当前页排序（与人员/设备列表保持一致）
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

async function clearAll() {
  await ElMessageBox.confirm('确认清空全部操作日志？该操作不可恢复。', '警告', { type: 'warning' });
  await operationLogApi.clear();
  // 中间件是在响应之后写日志的，所以「清空」这个动作本身也会留下一条记录 ——
  // 这是有意为之：审计日志被人清空这件事更值得留痕。这里明确提示，避免以为没清干净。
  ElMessage.success('已清空（本次清空操作本身会留下一条记录）');
  query.page = 1;
  loadData();
}

onMounted(loadData);
</script>

<style scoped lang="scss">
.log-tip {
  margin-bottom: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
.req-path {
  font-family: var(--el-font-family-mono, monospace);
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.deleted-user {
  color: var(--el-text-color-placeholder);
}
</style>
