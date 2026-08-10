<template>
  <el-card shadow="never" class="filter-panel">
    <div class="filters">
      <div class="filter-item">
        <span class="filter-label">厨具</span>
        <el-select
          v-model="filterStore.kitchenware"
          multiple
          collapse-tags
          collapse-tags-tooltip
          clearable
          placeholder="选择厨具"
          class="filter-select"
        >
          <el-option v-for="k in options.kitchenware" :key="k" :label="k" :value="k" />
        </el-select>
      </div>

      <div class="filter-item">
        <span class="filter-label">标签</span>
        <el-select
          v-model="filterStore.tags"
          multiple
          collapse-tags
          collapse-tags-tooltip
          clearable
          placeholder="选择标签"
          class="filter-select"
        >
          <el-option v-for="t in options.tags" :key="t" :label="t" :value="t" />
        </el-select>
      </div>

      <div class="filter-item">
        <span class="filter-label">食材</span>
        <el-select
          v-model="filterStore.ingredients"
          multiple
          collapse-tags
          collapse-tags-tooltip
          clearable
          placeholder="选择主要食材"
          class="filter-select"
        >
          <el-option v-for="i in options.ingredients" :key="i" :label="i" :value="i" />
        </el-select>
      </div>

      <el-button
        v-if="filterStore.isActive"
        type="danger"
        plain
        :icon="RefreshLeft"
        @click="filterStore.reset()"
      >
        重置
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { RefreshLeft } from '@element-plus/icons-vue'
import { useFilter } from '../composables/useFilter'
import { useFilterStore } from '../stores/filter'

const filterStore = useFilterStore()
const { options } = useFilter()
</script>

<style scoped>
.filter-panel {
  margin-bottom: 20px;
  border-radius: 12px;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 220px;
}

.filter-label {
  white-space: nowrap;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.filter-select {
  width: 100%;
}
</style>
