<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><User /></el-icon>
        人员材料
      </div>
    </div>

    <div class="mat-tip">
      <el-icon style="margin-right: 6px; vertical-align: -2px"><InfoFilled /></el-icon>
      选择需要办理的人员材料业务，系统将按人员档案自动填充资料 / 汇总原件。
    </div>

    <div class="mat-grid">
      <div
        v-for="item in items"
        :key="item.key"
        class="mat-card"
        :class="{ 'is-todo': item.todo }"
        tabindex="0"
        @click="onAction(item)"
        @keyup.enter="onAction(item)"
      >
        <div class="mat-card-top">
          <div class="mat-icon" :class="item.todo ? 'is-todo' : 'is-active'">
            <el-icon :size="28"><component :is="item.icon" /></el-icon>
          </div>
          <el-tag v-if="item.todo" size="small" type="info" effect="plain">待补充</el-tag>
          <el-tag v-else size="small" type="success" effect="plain">可用</el-tag>
        </div>
        <div class="mat-name">{{ item.name }}</div>
        <div class="mat-desc">{{ item.desc }}</div>
        <div class="mat-foot">
          <span class="mat-foot-text">{{ item.todo ? '功能建设中' : '点击进入' }}</span>
          <el-icon class="mat-arrow"><ArrowRight /></el-icon>
        </div>
      </div>
    </div>

    <!-- 人员证件批量下载 -->
    <CertExportDialog v-model="certVisible" />
    <!-- 劳动合同生成 -->
    <ContractGenDialog v-model="contractVisible" />
    <!-- 理工人员报名材料生成 -->
    <LigongEnrollDialog v-model="ligongVisible" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage } from 'element-plus';
import CertExportDialog from '../../components/materials/CertExportDialog.vue';
import ContractGenDialog from '../../components/materials/ContractGenDialog.vue';
import LigongEnrollDialog from '../../components/materials/LigongEnrollDialog.vue';

interface MatItem {
  key: string;
  name: string;
  icon: string;
  desc: string;
  todo: boolean;
}

const items: MatItem[] = [
  {
    key: 'cert',
    name: '人员证件批量下载',
    icon: 'Download',
    desc: '按所属检测站、在职状态批量打包下载安检证、环保证、身份证原件，并附缺证名单',
    todo: false,
  },
  {
    key: 'resume',
    name: '个人履历生成',
    icon: 'User',
    desc: '汇总人员基本信息与工作经历，生成标准化个人履历文档',
    todo: true,
  },
  {
    key: 'contract',
    name: '劳动合同生成',
    icon: 'DocumentCopy',
    desc: '按人员档案生成聘用劳动合同，支持在线预览与下载 Word 文件',
    todo: false,
  },
  {
    key: 'apply',
    name: '理工人员报名材料生成',
    icon: 'Tickets',
    desc: '选择报名人员，自动生成报名汇总表、安全责任书、证件原件与劳动合同，压缩包一键下载',
    todo: false,
  },
  {
    key: 'change',
    name: '理工人员变更材料生成',
    icon: 'RefreshLeft',
    desc: '生成理工人员变更申报所需材料',
    todo: true,
  },
];

const certVisible = ref(false);
const contractVisible = ref(false);
const ligongVisible = ref(false);

function onAction(item: MatItem) {
  if (item.key === 'cert') {
    certVisible.value = true;
    return;
  }
  if (item.key === 'contract') {
    contractVisible.value = true;
    return;
  }
  if (item.key === 'apply') {
    ligongVisible.value = true;
    return;
  }
  ElMessage.info(`「${item.name}」功能待补充，敬请期待`);
}
</script>

<style scoped lang="scss">
.mat-tip {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 10px 14px;
  margin-bottom: 16px;
}

.mat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.mat-card {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 18px 18px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-fill-color-blank);
  cursor: pointer;
  transition: all 0.22s ease;
  user-select: none;

  &:hover,
  &:focus-visible {
    border-color: var(--el-color-primary);
    box-shadow: 0 8px 22px rgba(64, 158, 255, 0.14);
    transform: translateY(-2px);
  }
  &:active {
    transform: translateY(0);
  }
}

.mat-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 12px;
}

.mat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.mat-icon.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.mat-icon.is-todo {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}

.mat-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.mat-desc {
  margin-top: 6px;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  flex: 1;
}

.mat-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.mat-arrow {
  color: var(--el-text-color-placeholder);
  transition: transform 0.22s ease;
}
.mat-card:hover .mat-arrow {
  transform: translateX(3px);
  color: var(--el-color-primary);
}

/* 待补充卡片：低调呈现，弱化可点暗示 */
.mat-card.is-todo {
  opacity: 0.86;
  &:hover,
  &:focus-visible {
    border-color: var(--el-border-color);
    box-shadow: none;
    transform: none;
  }
  .mat-name {
    color: var(--el-text-color-regular);
  }
}

@media screen and (max-width: 480px) {
  .mat-grid {
    grid-template-columns: 1fr;
  }
}
</style>
