<template>
  <div class="file-preview">
    <template v-if="!fileId">
      <el-empty description="无文件" :image-size="60" />
    </template>
    <template v-else-if="previewType === 'pdf'">
      <vue-pdf-embed :source="previewUrl" class="pdf-viewer" />
    </template>
    <template v-else-if="previewType === 'image'">
      <div class="image-scroll-wrap">
        <el-image :src="previewUrl" fit="contain" class="image-preview" :preview-src-list="[previewUrl]" />
      </div>
    </template>
    <template v-else-if="previewType === 'docx'">
      <div ref="docxContainer" class="docx-container"></div>
    </template>
    <template v-else>
      <div class="unsupported">
        <el-icon size="40"><Document /></el-icon>
        <p>该文件类型不支持在线预览</p>
        <el-button type="primary" size="small" @click="download">下载文件</el-button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue';
import VuePdfEmbed from 'vue-pdf-embed';
import { renderAsync } from 'docx-preview';
import { fileApi } from '../api/file';

const props = defineProps<{
  fileId: number;
  fileName?: string;
  mimeType?: string;
}>();

const previewUrl = computed(() => fileApi.previewUrl(props.fileId));
const docxContainer = ref<HTMLElement | null>(null);

const previewType = computed(() => {
  const mime = props.mimeType || '';
  const name = props.fileName || '';
  if (mime === 'application/pdf' || name.endsWith('.pdf')) return 'pdf';
  if (mime.startsWith('image/') || /\.(jpg|jpeg|png|gif|webp|bmp)$/i.test(name)) return 'image';
  if (
    mime === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' ||
    name.endsWith('.docx')
  )
    return 'docx';
  return 'unsupported';
});

watch(
  () => props.fileId,
  async () => {
    if (previewType.value === 'docx') {
      await nextTick();
      if (docxContainer.value) {
        try {
          const response = await fetch(previewUrl.value, {
            headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
          });
          const blob = await response.blob();
          await renderAsync(blob, docxContainer.value);
        } catch (e) {
          console.error('docx预览失败', e);
        }
      }
    }
  },
  { immediate: true }
);

function download() {
  window.open(fileApi.downloadUrl(props.fileId), '_blank');
}
</script>

<style scoped lang="scss">
.file-preview {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.image-scroll-wrap {
  width: 100%;
  max-height: 70vh;
  overflow: auto;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  background: #f5f5f5;
}
.image-preview {
  display: block;
  max-width: 100%;
  width: auto;
  height: auto;
}
.pdf-viewer {
  width: 100%;
  max-height: 600px;
  overflow: auto;
}
.docx-container {
  width: 100%;
  max-height: 600px;
  overflow: auto;
  :deep(.docx-wrapper) {
    background: #fff;
    padding: 10px;
  }
}
.unsupported {
  text-align: center;
  color: #909399;
  p {
    margin: 12px 0;
  }
}
@media screen and (max-width: 768px) {
  .pdf-viewer,
  .docx-container {
    max-height: 70vh;
  }
}
</style>
