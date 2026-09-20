<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><FolderOpened /></el-icon>
        数据备份
      </div>
      <el-button type="primary" :loading="running" @click="runNow">
        <el-icon><Download /></el-icon>立即备份
      </el-button>
    </div>

    <el-card shadow="never" class="bk-card">
      <div class="bk-form">
        <div class="bk-item">
          <span class="bk-label">自动备份</span>
          <el-switch v-model="form.enabled" />
          <span class="bk-hint">关闭后只保留手动「立即备份」</span>
        </div>
        <div class="bk-item">
          <span class="bk-label">备份间隔</span>
          <el-input-number v-model="form.intervalHours" :min="1" :max="720" :disabled="!form.enabled" />
          <span class="bk-hint">小时（24 = 每天一次）</span>
        </div>
        <div class="bk-item">
          <span class="bk-label">保留份数</span>
          <el-input-number v-model="form.retention" :min="1" :max="200" />
          <span class="bk-hint">超出后自动删除最旧的。每份约等于上传附件总量，份数别设太大</span>
        </div>
        <div class="bk-item bk-actions">
          <el-button type="primary" :loading="saving" @click="save">保存设置</el-button>
          <span class="bk-hint">
            <template v-if="settings.nextRunAt">下次备份：{{ fmtTime(settings.nextRunAt) }}</template>
            <template v-else-if="form.enabled">保存后服务端最长 10 分钟内开始首次备份</template>
          </span>
        </div>
      </div>

      <div class="bk-meta">
        <div>
          <span class="bk-meta-key">备份内容</span>
          数据库（一致性快照）+ data 目录下的密钥文件 + uploads 上传附件，打包为一个 zip
        </div>
        <div>
          <span class="bk-meta-key">存放位置</span>
          <code>{{ dir || '-' }}</code>
          <span class="bk-sep">·</span>
          上次备份：{{ settings.lastRunAt ? fmtTime(settings.lastRunAt) : '从未' }}
        </div>
      </div>
    </el-card>

    <el-table
      :data="list"
      v-loading="loading"
      stripe
      border
      :row-style="{ height: '48px' }"
      :cell-style="{ padding: '4px 4px' }"
      :header-cell-style="{ padding: '8px 4px', background: '#fafafa', color: '#333', whiteSpace: 'nowrap' }"
    >
      <el-table-column type="index" label="序号" width="55" align="center" />
      <el-table-column prop="name" label="备份文件" min-width="280" show-overflow-tooltip />
      <el-table-column label="大小" width="110" align="right">
        <template #default="{ row }">{{ fmtSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="生成时间" width="170" align="center">
        <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" align="center">
        <template #default="{ row }">
          <div style="display:flex;justify-content:center;gap:2px">
            <el-button size="small" type="primary" plain @click="download(row)">下载</el-button>
            <el-button size="small" type="danger" plain @click="remove(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
      <template #empty>
        <div class="bk-empty">
          还没有备份文件 —— 点右上角「立即备份」生成第一份，或打开自动备份等待服务端执行。
        </div>
      </template>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import dayjs from 'dayjs';
import { backupApi } from '../../api/backup';
import { decodeExportFileName, triggerFileDownload } from '../../utils/download';

const loading = ref(false);
const saving = ref(false);
const running = ref(false);
const list = ref<any[]>([]);
const dir = ref('');
const settings = ref<any>({});
const form = reactive({ enabled: false, intervalHours: 24, retention: 10 });

function fmtTime(t?: string) {
  if (!t) return '-';
  const d = dayjs(t);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : t;
}
function fmtSize(n?: number) {
  if (n === undefined || n === null) return '-';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}

async function loadAll() {
  loading.value = true;
  try {
    const [s, l] = (await Promise.all([backupApi.settings(), backupApi.list()])) as any[];
    settings.value = s?.data || {};
    Object.assign(form, {
      enabled: !!settings.value.enabled,
      intervalHours: settings.value.intervalHours || 24,
      retention: settings.value.retention || 10,
    });
    list.value = l?.data?.list || [];
    dir.value = l?.data?.dir || settings.value.dir || '';
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const res: any = await backupApi.saveSettings({ ...form });
    settings.value = res?.data || settings.value;
    ElMessage.success('设置已保存');
  } finally {
    saving.value = false;
  }
}

async function runNow() {
  running.value = true;
  try {
    await backupApi.run();
    ElMessage.success('备份完成');
    loadAll();
  } finally {
    running.value = false;
  }
}

async function download(row: any) {
  const res: any = await backupApi.download(row.name);
  const filename = decodeExportFileName(res.headers?.['content-disposition']) || row.name;
  triggerFileDownload(res.data as Blob, filename);
}

async function remove(row: any) {
  await ElMessageBox.confirm(`确认删除备份「${row.name}」？该操作不可恢复。`, '警告', { type: 'warning' });
  await backupApi.remove(row.name);
  ElMessage.success('已删除');
  loadAll();
}

onMounted(loadAll);
</script>

<style scoped lang="scss">
.bk-card {
  margin-bottom: 16px;
}
.bk-form {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 28px;
}
.bk-item {
  display: flex;
  align-items: center;
  gap: 10px;
}
.bk-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}
.bk-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.bk-actions {
  margin-left: auto;
}
.bk-meta {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed var(--el-border-color);
  font-size: 12.5px;
  line-height: 1.9;
  color: var(--el-text-color-secondary);
}
.bk-meta-key {
  display: inline-block;
  min-width: 60px;
  margin-right: 8px;
  color: var(--el-text-color-regular);
}
.bk-meta code {
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-size: 12px;
}
.bk-sep {
  margin: 0 8px;
}
.bk-empty {
  padding: 18px 0;
  color: var(--el-text-color-secondary);
}
@media screen and (max-width: 768px) {
  .bk-actions {
    margin-left: 0;
  }
}
</style>
