<template>
  <el-tag :type="info.type" size="small" effect="light">{{ info.label }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useExpiry, type ExpiryInfo } from '../composables/useExpiry';

const props = defineProps<{ expiryDate: string | Date | null }>();
const { calcExpiry } = useExpiry();
const info = computed<ExpiryInfo>(() => calcExpiry(props.expiryDate));
</script>
