<template>
  <el-dialog
    v-model="visible"
    title="人员证件批量下载"
    :width="isMobile ? '94%' : '620px'"
    align-center
    destroy-on-close
    @open="onOpen"
  >
    <div class="export-block">
      <div class="export-block-title">
        <el-icon><Filter /></el-icon>
        <span>1. 选择下载范围</span>
      </div>
      <div class="export-scope-grid">
        <div class="export-field">
          <div class="export-field-label">所属检测站</div>
          <el-select v-model="exportForm.department" placeholder="全部检测站" clearable style="width: 100%">
            <el-option v-for="d in departments" :key="d" :label="d" :value="d" />
          </el-select>
        </div>
        <div class="export-field">
          <div class="export-field-label">在职状态</div>
          <el-select v-model="exportForm.status" placeholder="全部状态" clearable style="width: 100%">
            <el-option label="在职" value="active" />
            <el-option label="离职" value="resigned" />
          </el-select>
        </div>
      </div>
    </div>

    <div class="export-block">
      <div class="export-block-title">
        <el-icon><Download /></el-icon>
        <span>2. 选择要打包下载的证件</span>
      </div>
      <div class="export-action-btns">
        <el-button
          v-for="act in exportActions"
          :key="act.name"
          :type="act.type"
          size="large"
          :loading="exportingCert === act.name"
          :disabled="exportingCert !== '' && exportingCert !== act.name"
          @click="downloadCertBatch(act.name)"
        >
          <el-icon><component :is="act.icon" /></el-icon>
          <span>一键下载{{ act.name }}</span>
        </el-button>
      </div>
      <p class="export-tip">
        将把「{{ exportScopeText }}」范围内所有人员持有的对应证件原件打包为 ZIP 下载；尚未上传该证件的人员会自动列入压缩包内缺证名单，便于跟进补办。
      </p>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue';
import { ElMessage } from 'element-plus';
import { personnelApi } from '../../api/personnel';
import { decodeExportFileName, triggerFileDownload } from '../../utils/download';

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    /** 初始范围（可选）：进入时预填的检测站 / 状态 */
    defaultDepartment?: string;
    defaultStatus?: string;
  }>(),
  { defaultDepartment: '', defaultStatus: '' }
);
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>();

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
});

const isMobile = ref(window.innerWidth <= 768);
function syncIsMobile() {
  isMobile.value = window.innerWidth <= 768;
}
window.addEventListener('resize', syncIsMobile);
onUnmounted(() => window.removeEventListener('resize', syncIsMobile));

const departments = ref<string[]>([]);
const exportForm = reactive({ department: '', status: '' });
const exportingCert = ref('');
const exportActions: Array<{ name: string; icon: string; type: 'primary' | 'success' | 'warning' }> = [
  { name: '安检证', icon: 'Document', type: 'primary' },
  { name: '环保证', icon: 'Files', type: 'success' },
  { name: '身份证', icon: 'Postcard', type: 'warning' },
];

const exportScopeText = computed(() => {
  const dept = exportForm.department || '全部检测站';
  const st = exportForm.status === 'active' ? '在职' : exportForm.status === 'resigned' ? '离职' : '全部状态';
  return `${dept} · ${st}`;
});

async function loadOptions() {
  try {
    const res: any = await personnelApi.options();
    departments.value = res.data.departments || [];
  } catch {
    /* 选项加载失败不影响下载 */
  }
}

function onOpen() {
  // 每次打开同步初始范围（可跟随调用方传入的筛选条件）
  exportForm.department = props.defaultDepartment || '';
  exportForm.status = props.defaultStatus || '';
  if (!departments.value.length) loadOptions();
}

async function downloadCertBatch(certName: string) {
  if (exportingCert.value) return;
  exportingCert.value = certName;
  try {
    const res: any = await personnelApi.exportCerts({
      department: exportForm.department,
      status: exportForm.status,
      certName,
    });
    const headers = res.headers || {};
    const total = Number(headers['x-export-total'] || 0);
    const skipped = Number(headers['x-export-skipped'] || 0);
    const filename = decodeExportFileName(headers['content-disposition']) || `${certName}批量下载.zip`;
    triggerFileDownload(res.data as Blob, filename);
    let msg = `已打包下载 ${certName} ${total} 份（${exportScopeText.value}）`;
    if (skipped > 0) msg += `；${skipped} 人未上传，缺证名单已写入压缩包`;
    ElMessage.success(msg);
  } catch {
    // 后端错误已由请求拦截器统一提示
  } finally {
    exportingCert.value = '';
  }
}

onMounted(() => {
  loadOptions();
});
</script>

<style scoped lang="scss">
.export-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 14px 16px 16px;
  margin-bottom: 14px;
  background: var(--el-fill-color-blank);

  &:last-child {
    margin-bottom: 0;
  }
}
.export-block-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;

  .el-icon {
    color: var(--el-color-primary);
    font-size: 16px;
  }
}
.export-scope-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.export-field-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}
.export-action-btns {
  display: flex;
  flex-direction: column;
  gap: 10px;

  .el-button {
    margin-left: 0;
    width: 100%;
    justify-content: center;
    font-weight: 600;
  }
}
.export-tip {
  margin: 12px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 8px 10px;
}
@media screen and (max-width: 480px) {
  .export-scope-grid {
    grid-template-columns: 1fr;
  }
}
</style>
