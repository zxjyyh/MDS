<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-title">
        <el-icon style="margin-right: 6px; vertical-align: -2px"><Connection /></el-icon>
        IP信息
      </div>
    </div>

    <el-card shadow="never" class="ip-card">
      <el-empty description="IP信息模块建设中，敬请期待">
        <p class="ip-tip">该模块的具体功能后续补充。</p>
      </el-empty>
    </el-card>
  </div>
</template>

<script setup lang="ts">
// IP信息 —— 功能占位页
// TODO: 后续在此实现 IP 信息的维护与查询
</script>

<style scoped lang="scss">
.ip-card {
  min-height: calc(100vh - 180px);
  display: flex;
  align-items: center;
  justify-content: center;
}
.ip-card :deep(.el-card__body) {
  width: 100%;
}
.ip-tip {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
