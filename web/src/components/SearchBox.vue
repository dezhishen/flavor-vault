<template>
  <el-input
    v-model="text"
    class="search-box"
    size="large"
    clearable
    :prefix-icon="Search"
    placeholder="搜索菜谱、食材、厨具、步骤…"
    aria-label="搜索菜谱"
    @clear="apply('')"
    @keyup.enter="apply(text)"
  />
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { useFilterStore } from '../stores/filter'

const filterStore = useFilterStore()

// 本地输入值（与全局关键词解耦，便于防抖）
const text = ref(filterStore.keyword)
let timer: number | undefined

function apply(v: string) {
  filterStore.keyword = v
}

watch(text, (v) => {
  if (timer) window.clearTimeout(timer)
  // 300ms 防抖，避免每次击键都重建结果
  timer = window.setTimeout(() => apply(v ?? ''), 300)
})

onBeforeUnmount(() => {
  if (timer) window.clearTimeout(timer)
})
</script>

<style scoped>
.search-box {
  margin-bottom: 16px;
}
.search-box :deep(.el-input__wrapper) {
  border-radius: 20px;
}
</style>
