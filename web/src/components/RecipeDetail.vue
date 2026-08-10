<template>
  <el-card shadow="never" class="recipe-detail" :body-style="{ padding: '24px' }">
    <!-- 头部 -->
    <div class="detail-head">
      <div class="detail-title">{{ detail.name }}</div>
      <div class="detail-tags">
        <el-tag v-for="t in detail.tags" :key="t" type="warning" effect="plain" class="tag">
          {{ t }}
        </el-tag>
        <el-tag v-for="k in detail.kitchenware" :key="k" type="info" effect="plain" class="tag">
          🔧 {{ k }}
        </el-tag>
      </div>
      <div v-if="detail.description" class="detail-desc">{{ detail.description }}</div>
    </div>

    <el-descriptions :column="4" border class="detail-stats" size="small">
      <el-descriptions-item label="准备时间">{{ detail.stats.prep_time }} 分钟</el-descriptions-item>
      <el-descriptions-item label="烹饪时间">{{ detail.stats.cook_time }} 分钟</el-descriptions-item>
      <el-descriptions-item label="难度">
        <el-rate :model-value="detail.stats.difficulty" disabled />
      </el-descriptions-item>
      <el-descriptions-item label="总耗时">
        {{ detail.stats.prep_time + detail.stats.cook_time }} 分钟
      </el-descriptions-item>
    </el-descriptions>

    <!-- 食材 -->
    <el-divider content-position="left">食材</el-divider>
    <div class="ingredients">
      <div v-if="detail.ingredients.main.length" class="ing-group">
        <div class="ing-group-title">主要食材</div>
        <el-table :data="detail.ingredients.main" size="small" border>
          <el-table-column prop="name" label="食材" min-width="120" />
          <el-table-column prop="amount" label="用量" min-width="120" />
        </el-table>
      </div>
      <div v-if="detail.ingredients.side.length" class="ing-group">
        <div class="ing-group-title">配菜 / 辅料</div>
        <el-table :data="detail.ingredients.side" size="small" border>
          <el-table-column prop="name" label="食材" min-width="120" />
          <el-table-column prop="amount" label="用量" min-width="120" />
        </el-table>
      </div>
    </div>

    <!-- 步骤 -->
    <el-divider content-position="left">步骤</el-divider>
    <el-steps direction="vertical" :active="detail.steps.length">
      <el-step
        v-for="s in detail.steps"
        :key="s.order"
        :title="`步骤 ${s.order}`"
        :description="s.description"
      />
    </el-steps>

    <!-- 图片 / 视频 -->
    <template v-if="(detail.media.images?.length || 0) || detail.media.video_url">
      <el-divider content-position="left">过程图</el-divider>
      <div class="images">
        <el-image
          v-for="(img, i) in detail.media.images || []"
          :key="i"
          :src="resolveAsset(img)"
          fit="cover"
          class="step-img"
          :preview-src-list="(detail.media.images || []).map(resolveAsset)"
        />
      </div>
      <div v-if="detail.media.video_url" class="video">
        <el-link type="primary" :href="detail.media.video_url" target="_blank">
          查看视频
        </el-link>
      </div>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import type { RecipeDetail } from '../types'

defineProps<{ detail: RecipeDetail }>()

function resolveAsset(p: string): string {
  if (/^(https?:)?\/\//.test(p)) return p
  return `./assets/${p}`
}
</script>

<style scoped>
.recipe-detail {
  border-radius: 12px;
}

.detail-title {
  font-size: 26px;
  font-weight: 700;
  margin-bottom: 8px;
}

.detail-tags .tag {
  margin-right: 8px;
  margin-bottom: 4px;
}

.detail-desc {
  color: var(--el-text-color-secondary);
  margin-top: 8px;
  line-height: 1.6;
}

.detail-stats {
  margin-top: 16px;
}

.ing-group {
  margin-bottom: 16px;
}

.ing-group-title {
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}

.images {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.step-img {
  width: 180px;
  height: 120px;
  border-radius: 8px;
}

.video {
  margin-top: 12px;
}
</style>
