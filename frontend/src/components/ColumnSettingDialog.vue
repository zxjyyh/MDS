<template>
  <el-dialog
    :model-value="modelValue"
    title="展示设置"
    :width="isMobile ? '92%' : '520px'"
    align-center
    destroy-on-close
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <div class="col-tip">
      <el-icon style="margin-right: 6px; vertical-align: -2px"><InfoFilled /></el-icon>
      勾选需要在列表中展示的字段（序号与操作列始终显示）。设置按当前账号保存，下次进入自动沿用。
      标了「默认隐藏」的字段默认不出现在列表里，勾选后才会显示。
    </div>

    <el-checkbox-group v-model="checked" class="col-grid">
      <el-checkbox v-for="c in columns" :key="c.prop" :value="c.prop">
        {{ c.label }}<span v-if="c.defaultHidden" class="col-default-hide">默认隐藏</span>
      </el-checkbox>
    </el-checkbox-group>

    <div class="col-count">已选 {{ checked.length }} / {{ columns.length }} 项</div>

    <template #footer>
      <el-button @click="emit('reset')">恢复默认</el-button>
      <el-button type="primary" @click="emit('update:modelValue', false)">完成</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue';
import type { ColumnDef } from '../composables/useColumnSetting';

const props = defineProps<{
  modelValue: boolean;
  columns: ColumnDef[];
  visible: string[];
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void;
  (e: 'update:visible', v: string[]): void;
  (e: 'reset'): void;
}>();

const checked = computed({
  get: () => props.visible,
  set: (v: string[]) => emit('update:visible', v),
});

const isMobile = ref(window.innerWidth <= 768);
function syncIsMobile() {
  isMobile.value = window.innerWidth <= 768;
}
window.addEventListener('resize', syncIsMobile);
onUnmounted(() => window.removeEventListener('resize', syncIsMobile));
</script>

<style scoped lang="scss">
.col-tip {
  display: flex;
  align-items: center;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 8px 12px;
  margin-bottom: 14px;
}
.col-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 6px 12px;
  align-items: start;
}
/* 标了 defaultHidden 的字段给个小角标：这些列默认不出现，很容易找不到 */
.col-default-hide {
  margin-left: 4px;
  padding: 0 3px;
  font-size: 10.5px;
  line-height: 1.5;
  color: var(--el-color-warning);
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-7);
  border-radius: 3px;
  white-space: nowrap;
}
.col-count {
  margin-top: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
