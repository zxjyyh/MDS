<template>
  <el-tree
    ref="treeRef"
    :data="treeData"
    :props="{ label: 'name', children: 'children' }"
    node-key="id"
    :default-expand-all="true"
    :highlight-current="true"
    :expand-on-click-node="false"
    @node-click="handleNodeClick"
  >
    <template #default="{ node, data }">
      <span class="tree-node">
        <el-icon><Folder /></el-icon>
        <span class="node-label">{{ node.label }}</span>
        <span v-if="showActions" class="node-actions" @click.stop>
          <el-button text size="small" @click="emit('add', data)">
            <el-icon><Plus /></el-icon>
          </el-button>
          <el-button text size="small" @click="emit('edit', data)">
            <el-icon><Edit /></el-icon>
          </el-button>
          <el-button text size="small" type="danger" @click="emit('remove', data)">
            <el-icon><Delete /></el-icon>
          </el-button>
        </span>
      </span>
    </template>
  </el-tree>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { categoryApi } from '../api/document';

const props = defineProps<{
  systemType: string;
  showActions?: boolean;
}>();

const emit = defineEmits<{
  select: [node: any];
  add: [parent: any];
  edit: [node: any];
  remove: [node: any];
}>();

const treeRef = ref();
const treeData = ref<any[]>([]);

async function loadTree() {
  const res: any = await categoryApi.getTree(props.systemType);
  treeData.value = res.data || [];
}

function handleNodeClick(node: any) {
  emit('select', node);
}

defineExpose({ loadTree, treeRef });

watch(() => props.systemType, loadTree, { immediate: true });
</script>

<style scoped lang="scss">
.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  .node-label {
    margin-left: 4px;
  }
  .node-actions {
    margin-left: auto;
    display: none;
  }
  &:hover .node-actions {
    display: flex;
  }
}
</style>
