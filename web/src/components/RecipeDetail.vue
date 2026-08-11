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

    <!-- 多版本切换 -->
    <el-tabs v-if="versions.length > 1" v-model="activeIdx" class="version-tabs">
      <el-tab-pane
        v-for="(v, i) in versions"
        :key="i"
        :label="v.name || `版本 ${i + 1}`"
        :name="String(i)"
      />
    </el-tabs>

    <template v-if="activeVersion">
      <el-descriptions :column="isMobile ? 2 : 4" border class="detail-stats" size="small">
        <el-descriptions-item label="准备时间">{{ activeVersion.stats.prep_time }} 分钟</el-descriptions-item>
        <el-descriptions-item label="烹饪时间">{{ activeVersion.stats.cook_time }} 分钟</el-descriptions-item>
        <el-descriptions-item label="难度">
          <el-rate :model-value="activeVersion.stats.difficulty" disabled />
        </el-descriptions-item>
        <el-descriptions-item label="总耗时">
          {{ activeVersion.stats.prep_time + activeVersion.stats.cook_time }} 分钟
        </el-descriptions-item>
      </el-descriptions>

      <!-- 食材（必选 / 配菜 / 非必须） -->
      <el-divider content-position="left">食材</el-divider>
      <div class="ingredients">
        <div v-if="(activeVersion.ingredients.main || []).length" class="ing-group">
          <div class="ing-group-title">主要食材（必选）</div>
          <el-table :data="activeVersion.ingredients.main || []" size="small" border>
            <el-table-column prop="name" label="食材" min-width="120" />
            <el-table-column prop="amount" label="用量" min-width="100" />
            <el-table-column prop="note" label="备注" min-width="120" />
          </el-table>
        </div>
        <div v-if="(activeVersion.ingredients.side || []).length" class="ing-group">
          <div class="ing-group-title">配菜 / 辅料</div>
          <el-table :data="activeVersion.ingredients.side || []" size="small" border>
            <el-table-column prop="name" label="食材" min-width="120" />
            <el-table-column prop="amount" label="用量" min-width="100" />
            <el-table-column prop="note" label="备注" min-width="120" />
          </el-table>
        </div>
        <div v-if="(activeVersion.ingredients.optional || []).length" class="ing-group">
          <div class="ing-group-title">非必须（可选）</div>
          <el-table :data="activeVersion.ingredients.optional || []" size="small" border>
            <el-table-column prop="name" label="食材" min-width="120" />
            <el-table-column prop="amount" label="用量" min-width="100" />
            <el-table-column prop="note" label="备注" min-width="120" />
          </el-table>
        </div>
      </div>

      <!-- 调料（含备选方案） -->
      <template v-if="(activeVersion.seasonings || []).length">
        <el-divider content-position="left">调料</el-divider>
        <div class="seasonings">
          <div v-for="(s, i) in activeVersion.seasonings || []" :key="i" class="seasoning-item">
            <span class="seasoning-name">方案一：{{ s.name }}</span>
            <span v-if="s.amount" class="seasoning-amount">{{ s.amount }}</span>
            <span v-if="s.note" class="seasoning-note">{{ s.note }}</span>
            <div v-if="(s.alternatives || []).length" class="seasoning-alts">
              <el-tag
                v-for="(a, j) in s.alternatives || []"
                :key="j"
                size="small"
                type="success"
                effect="plain"
                class="alt-tag"
              >
                方案{{ j + 2 }}：{{ a.name }}{{ a.amount ? ` ${a.amount}` : '' }}{{ a.note ? `（${a.note}）` : '' }}
              </el-tag>
            </div>
          </div>
        </div>
      </template>

      <!-- 步骤 -->
      <el-divider content-position="left">步骤</el-divider>
      <el-steps direction="vertical" :active="(activeVersion.steps || []).length">
        <el-step v-for="s in activeVersion.steps || []" :key="s.order" :title="`第 ${s.order} 步`">
          <template #description>
            <div class="step-desc">{{ s.description }}</div>
            <el-image
              v-if="s.image_ref"
              :src="resolveAsset(s.image_ref)"
              fit="cover"
              class="step-img"
              :preview-src-list="[resolveAsset(s.image_ref)]"
              :preview-teleported="true"
            />
          </template>
        </el-step>
      </el-steps>

      <!-- 图片 / 视频 -->
      <template v-if="(activeVersion.media.images?.length || 0) || activeVersion.media.video_url">
        <el-divider content-position="left">过程图</el-divider>
        <div class="images">
          <el-image
            v-for="(img, i) in activeVersion.media.images || []"
            :key="i"
            :src="resolveAsset(img)"
            fit="cover"
            class="step-img"
            :preview-src-list="(activeVersion.media.images || []).map(resolveAsset)"
          />
        </div>
        <div v-if="activeVersion.media.video_url" class="video">
          <el-link type="primary" :href="activeVersion.media.video_url" target="_blank">
            查看视频
          </el-link>
        </div>
      </template>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useViewport } from '../composables/useViewport'
import type { RecipeDetail, Version } from '../types'

const props = defineProps<{ detail: RecipeDetail }>()

// 移动端描述列数减少，避免挤压
const { isMobile } = useViewport()

// 多版本：versions 非空用 versions；否则顶层字段作为单个默认版本
const versions = computed<Version[]>(() => {
  if (props.detail.versions?.length) return props.detail.versions
  return [
    {
      name: '默认',
      ingredients: props.detail.ingredients,
      seasonings: props.detail.seasonings || [],
      steps: props.detail.steps,
      media: props.detail.media,
      stats: props.detail.stats,
    },
  ]
})

const activeIdx = ref('0')
const activeVersion = computed<Version>(
  () => versions.value[Number(activeIdx.value)] ?? versions.value[0]
)

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

.step-desc {
  line-height: 1.6;
  margin-bottom: 8px;
  white-space: pre-wrap;
}

/* 步骤配图（略小于过程图） */
.el-step .step-img {
  width: 160px;
  height: 110px;
  border-radius: 8px;
}

.step-img {
  width: 180px;
  height: 120px;
  border-radius: 8px;
}

.video {
  margin-top: 12px;
}

.version-tabs {
  margin-top: 4px;
}

.seasoning-item {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.seasoning-name {
  font-weight: 600;
}

.seasoning-amount,
.seasoning-note {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.seasoning-alts {
  display: inline-flex;
  gap: 6px;
  flex-wrap: wrap;
}

/* H5 自适应 */
@media (max-width: 768px) {
  .recipe-detail :deep(.el-card__body) {
    padding: 14px !important;
  }
  .detail-title {
    font-size: 21px;
  }
  .detail-desc {
    font-size: 13px;
  }
  .el-step .step-img,
  .step-img {
    width: 120px;
    height: 90px;
  }
  .step-desc {
    font-size: 13px;
  }
  /* 表格横向可滚动，避免溢出 */
  .ingredients :deep(.el-table) {
    width: 100%;
    overflow-x: auto;
    display: block;
  }
}
</style>
