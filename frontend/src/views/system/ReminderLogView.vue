<template>
  <div class="page-container">
    <div class="page-header">
      <span class="page-title">提醒日志</span>
      <el-button type="primary" plain @click="testRun" :loading="testing">手动触发检查</el-button>
    </div>

    <div class="search-bar">
      <el-select v-model="query.module" placeholder="模块" clearable style="width: 160px" @change="loadData">
        <el-option label="人员证件" value="personnel_cert" />
        <el-option label="设备证书" value="equipment_cert" />
        <el-option label="体系文档" value="document" />
      </el-select>
      <el-button type="primary" @click="loadData">查询</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe border>
      <el-table-column type="index" label="#" width="50" />
      <el-table-column label="模块" width="110">
        <template #default="{ row }">{{ moduleLabel(row.module) }}</template>
      </el-table-column>
      <el-table-column prop="targetName" label="名称" min-width="200" show-overflow-tooltip />
      <el-table-column label="到期日期" width="110">
        <template #default="{ row }">{{ row.expiryDate ? formatDate(row.expiryDate) : '-' }}</template>
      </el-table-column>
      <el-table-column label="提前天数" width="90" align="center">
        <template #default="{ row }">{{ row.daysBefore != null ? row.daysBefore + '天' : '-' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 'sent' ? 'success' : 'danger'" size="small">
            {{ row.status === 'sent' ? '已发送' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="recipients" label="收件人" min-width="180" show-overflow-tooltip />
      <el-table-column prop="errorMsg" label="错误信息" min-width="180" show-overflow-tooltip />
      <el-table-column label="发送时间" width="160">
        <template #default="{ row }">{{ formatDateTime(row.sentAt) }}</template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :total="total" layout="total, prev, pager, next" @current-change="loadData" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { reminderApi } from '../../api/dashboard';
import dayjs from 'dayjs';

const list = ref<any[]>([]);
const loading = ref(false);
const total = ref(0);
const query = reactive({ page: 1, pageSize: 20, module: '' });
const testing = ref(false);

function moduleLabel(m: string) {
  return { personnel_cert: '人员证件', equipment_cert: '设备证书', document: '体系文档' }[m] || m;
}
function formatDate(d: string) { return dayjs(d).format('YYYY-MM-DD'); }
function formatDateTime(d: string) { return dayjs(d).format('YYYY-MM-DD HH:mm:ss'); }

async function loadData() {
  loading.value = true;
  try {
    const res: any = await reminderApi.logs(query);
    list.value = res.data.list;
    total.value = res.data.total;
  } finally { loading.value = false; }
}

async function testRun() {
  testing.value = true;
  try {
    await reminderApi.testRun();
    ElMessage.success('已触发到期检查任务');
    setTimeout(loadData, 3000);
  } finally { testing.value = false; }
}

onMounted(loadData);
</script>
