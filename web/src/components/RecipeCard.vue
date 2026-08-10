<template>
  <el-card
    shadow="hover"
    class="recipe-card"
    :body-style="{ padding: '0' }"
    @click="$emit('click')"
  >
    <div class="card-cover">
      <el-image
        v-if="recipe.cover"
        :src="resolveImage(recipe.cover)"
        fit="cover"
        class="cover-img"
      >
        <template #error>
          <div class="cover-fallback">🍽️</div>
        </template>
      </el-image>
      <div v-else class="cover-fallback">🍽️</div>
      <span class="difficulty-badge">难度 ★{{ recipe.difficulty }}</span>
    </div>

    <div class="card-body">
      <div class="card-title">{{ recipe.name }}</div>
      <div class="card-desc">{{ recipe.description || '暂无简介' }}</div>
      <div class="card-tags">
        <el-tag
          v-for="t in recipe.tags.slice(0, 3)"
          :key="t"
          size="small"
          type="warning"
          effect="plain"
          class="tag"
        >
          {{ t }}
        </el-tag>
      </div>
      <div class="card-meta">
        <span class="meta-item">⏱ {{ recipe.prep_time + recipe.cook_time }} 分钟</span>
        <span v-if="recipe.kitchenware.length" class="meta-item">
          🔧 {{ recipe.kitchenware.join('、') }}
        </span>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { RecipeSummary } from '../types'

defineProps<{ recipe: RecipeSummary }>()
defineEmits<{ click: [] }>()

// 封面图：数据中为相对路径，这里解析到 assets 目录
function resolveImage(cover: string): string {
  if (/^(https?:)?\/\//.test(cover)) return cover
  return `./assets/${cover}`
}
</script>

<style scoped>
.recipe-card {
  cursor: pointer;
  border-radius: 12px;
  overflow: hidden;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.recipe-card:hover {
  transform: translateY(-3px);
}

.card-cover {
  position: relative;
  height: 140px;
  background: var(--el-fill-color-light);
}

.cover-img {
  width: 100%;
  height: 100%;
  display: block;
}

.cover-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
  color: var(--el-text-color-placeholder);
}

.difficulty-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
}

.card-body {
  padding: 12px 14px 14px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}

.card-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  min-height: 38px;
}

.card-tags {
  margin: 8px 0;
}

.tag {
  margin-right: 6px;
}

.card-meta {
  font-size: 12px;
  color: var(--el-text-color-regular);
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
</style>
