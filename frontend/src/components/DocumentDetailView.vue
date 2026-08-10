<template>
  <div class="page-container" v-loading="loading">
    <div class="page-header">
      <div>
        <el-button text @click="$router.back()"><el-icon><ArrowLeft /></el-icon> 返回</el-button>
        <span class="page-title" style="margin-left: 8px">{{ detail?.title }}</span>
      </div>
      <div>
        <el-button @click="downloadFile"><el-icon><Download /></el-icon> 下载</el-button>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :xs="24" :sm="6">
        <el-card shadow="never">
          <template #header><span>文档信息</span></template>
          <div class="info-list">
            <div class="info-item"><span class="label">标题</span><span>{{ detail?.title }}</span></div>
            <div class="info-item"><span class="label">编号</span><span>{{ detail?.docNumber || '-' }}</span></div>
            <div class="info-item"><span class="label">版本</span><span>{{ detail?.version || '-' }}</span></div>
            <div class="info-item"><span class="label">分类</span><span>{{ detail?.category?.name || '-' }}</span></div>
            <div class="info-item"><span class="label">发布日期</span><span>{{ formatDate(detail?.issueDate) }}</span></div>
            <div class="info-item"><span class="label">到期日期</span><span>{{ formatDate(detail?.expiryDate) }}</span></div>
            <div class="info-item"><span class="label">状态</span>
              <ExpiryBadge v-if="detail" :expiry-date="detail.expiryDate" />
            </div>
            <div class="info-item"><span class="label">上传人</span><span>{{ detail?.createdBy?.realName || '-' }}</span></div>
            <div class="info-item"><span class="label">上传时间</span><span>{{ detail ? formatDateTime(detail.createdAt) : '-' }}</span></div>
            <div class="info-item" v-if="detail?.description">
              <span class="label">描述</span><span>{{ detail.description }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="18">
        <el-card shadow="never">
          <template #header><span>文档预览</span></template>
          <FilePreview
            v-if="detail?.file"
            :file-id="detail.file.id"
            :file-name="detail.file.originalName"
            :mime-type="detail.file.mimeType"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { documentApi } from '../api/document';
import { fileApi } from '../api/file';
import ExpiryBadge from './ExpiryBadge.vue';
import FilePreview from './FilePreview.vue';
import dayjs from 'dayjs';

const route = useRoute();
const id = Number(route.params.id);

const detail = ref<any>(null);
const loading = ref(false);

function formatDate(d?: string) { return d ? dayjs(d).format('YYYY-MM-DD') : '-'; }
function formatDateTime(d?: string) { return d ? dayjs(d).format('YYYY-MM-DD HH:mm') : '-'; }

async function loadData() {
  loading.value = true;
  try {
    const res: any = await documentApi.detail(id);
    detail.value = res.data;
  } finally { loading.value = false; }
}

function downloadFile() {
  if (detail.value?.fileId) {
    window.open(fileApi.downloadUrl(detail.value.fileId), '_blank');
  }
}

onMounted(loadData);
</script>

<style scoped lang="scss">
.info-list {
  .info-item {
    display: flex;
    padding: 8px 0;
    border-bottom: 1px solid #f0f0f0;
    .label { width: 80px; color: #909399; flex-shrink: 0; }
  }
}
</style>
