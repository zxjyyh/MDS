<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><Setting /></el-icon>
        设备材料
      </div>
    </div>

    <el-card shadow="never" class="equip-card">
      <el-empty description="设备材料生成功能建设中，敬请期待">
        <p class="equip-tip">
          后续将支持按设备档案生成检定 / 校准记录、设备履历表、资质附件等设备类材料。
        </p>
      </el-empty>
    </el-card>
  </div>
</template>

<script setup lang="ts">
// 设备材料 —— 功能占位页
// TODO: 后续在此实现设备类材料生成（检定记录 / 设备履历 / 资质附件等）
</script>

<style scoped lang="scss">
.equip-card {
  min-height: calc(100vh - 180px);
  display: flex;
  align-items: center;
  justify-content: center;
}
.equip-card :deep(.el-card__body) {
  width: 100%;
}
.equip-tip {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
