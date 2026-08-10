<template>
  <el-upload
    ref="uploadRef"
    :show-file-list="false"
    :http-request="customUpload"
    :before-upload="beforeUpload"
    :accept="accept"
  >
    <el-button type="primary" :loading="uploading">
      <el-icon><Upload /></el-icon>
      {{ uploading ? `上传中 ${progress}%` : '上传文件' }}
    </el-button>
    <template #tip>
      <div class="upload-tip">{{ tip || '支持 PDF/Word/图片，最大 50MB' }}</div>
    </template>
  </el-upload>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage, type UploadRequestOptions } from 'element-plus';
import { fileApi } from '../api/file';

const props = defineProps<{
  accept?: string;
  tip?: string;
}>();

const emit = defineEmits<{ success: [file: any]; }>();

const uploadRef = ref();
const uploading = ref(false);
const progress = ref(0);

function beforeUpload(file: File) {
  const maxSize = 50 * 1024 * 1024;
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过 50MB');
    return false;
  }
  return true;
}

async function customUpload(options: UploadRequestOptions) {
  uploading.value = true;
  progress.value = 0;
  try {
    const res: any = await fileApi.upload(options.file as File, (p) => {
      progress.value = p;
    });
    ElMessage.success('上传成功');
    emit('success', res.data);
  } catch (e) {
    ElMessage.error('上传失败');
  } finally {
    uploading.value = false;
  }
}
</script>

<style scoped>
.upload-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
