<template>
  <el-card shadow="never" class="recipe-detail" :body-style="{ padding: '18px' }">
    <!-- 头部 -->
    <div class="detail-head">
      <div class="detail-head-top">
        <div class="detail-title">{{ detail.name }}</div>
        <el-button size="small" type="primary" plain :loading="capturing" @click="captureShot" class="shot-btn">
          <el-icon v-if="!capturing"><Download /></el-icon>
          <span>{{ capturing ? '生成中…' : '截图' }}</span>
        </el-button>
      </div>
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
      <div class="detail-layout">
        <!-- 主列：步骤优先 -->
        <div class="detail-main">
          <!-- 步骤 -->
          <div class="block steps-block order-4">
            <div class="block-title">📋 步骤</div>
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
          </div>

          <!-- 过程图 / 视频 -->
          <div
            v-if="(activeVersion.media.images?.length || 0) || activeVersion.media.video_url"
            class="block media-block order-6"
          >
            <div class="block-title">🖼 过程图 / 视频</div>
            <div class="images">
              <el-image
                v-for="(img, i) in activeVersion.media.images || []"
                :key="i"
                :src="resolveAsset(img)"
                fit="cover"
                class="gallery-img"
                :preview-src-list="(activeVersion.media.images || []).map(resolveAsset)"
              />
            </div>
            <div v-if="activeVersion.media.video_url" class="video">
              <el-link type="primary" :href="activeVersion.media.video_url" target="_blank">
                查看视频
              </el-link>
            </div>
          </div>
        </div>

        <!-- 侧栏：主要食材 / 调料 / 统计（桌面 sticky；移动端按 order 融入单列） -->
        <aside class="detail-side">
          <!-- 主要食材（非必须通过条目 Note 备注表达，无独立 optional 分组） -->
          <div v-if="mainIngredients.length" class="block main-ing-block order-1">
            <div class="block-title">🥘 主要食材</div>
            <IngredientTable :items="mainIngredients" />
          </div>

          <!-- 配菜 / 辅料（用料） -->
          <div v-if="(activeVersion.ingredients.side || []).length" class="block side-ing-block order-2">
            <div class="block-title">🥬 配菜 / 辅料</div>
            <IngredientTable :items="activeVersion.ingredients.side || []" />
          </div>

          <!-- 调料（含备选方案） -->
          <div v-if="(activeVersion.seasonings || []).length" class="block season-block order-3">
            <div class="block-title">🧂 调料</div>
            <IngredientTable :items="activeVersion.seasonings || []" name-label="调料" />
          </div>

          <!-- 统计卡（时间/难度；桌面放食材下方；移动端默认折叠藏起，顶部让给食材） -->
          <el-collapse v-model="statsActive" class="stats-collapse order-5">
            <el-collapse-item title="⏱ 时间 / 难度 / 总耗时" name="stats">
              <div class="stats-card">
                <div class="stat-item">
                  <span class="stat-icon">⏱</span>
                  <span class="stat-label">准备</span>
                  <span class="stat-value">{{ activeVersion.stats.prep_time }} 分钟</span>
                </div>
                <div class="stat-item">
                  <span class="stat-icon">🔥</span>
                  <span class="stat-label">烹饪</span>
                  <span class="stat-value">{{ activeVersion.stats.cook_time }} 分钟</span>
                </div>
                <div class="stat-item">
                  <span class="stat-icon">⭐</span>
                  <span class="stat-label">难度</span>
                  <span class="stat-value">
                    <el-rate :model-value="activeVersion.stats.difficulty" disabled size="small" />
                  </span>
                </div>
                <div class="stat-item">
                  <span class="stat-icon">🕐</span>
                  <span class="stat-label">总耗时</span>
                  <span class="stat-value">{{ activeVersion.stats.prep_time + activeVersion.stats.cook_time }} 分钟</span>
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>
        </aside>
      </div>
    </template>
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { toPng } from 'html-to-image'
import { ElMessage } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import { useViewport } from '../composables/useViewport'
import IngredientTable from './IngredientTable.vue'
import type { RecipeDetail, Version } from '../types'

const props = defineProps<{ detail: RecipeDetail }>()

const capturing = ref(false)

// 生成当前菜谱长图（PNG）并触发下载
async function captureShot() {
  const node = document.querySelector<HTMLElement>('.recipe-detail')
  if (!node) return
  capturing.value = true
  try {
    const dataUrl = await toPng(node, {
      pixelRatio: 2,
      backgroundColor: '#ffffff',
      cacheBust: true,
    })
    const link = document.createElement('a')
    link.download = `${props.detail.name || '菜谱'}.png`
    link.href = dataUrl
    link.click()
    ElMessage.success('长图已生成，开始下载')
  } catch (e) {
    console.error(e)
    ElMessage.error('截图失败（可能含跨域图片），请稍后重试')
  } finally {
    capturing.value = false
  }
}

// 统计卡（时间/难度）：桌面默认展开，移动端默认折叠藏起（顶部让给食材）
const { isMobile } = useViewport()
const statsActive = ref<string[]>(['stats'])
if (isMobile.value) statsActive.value = []
watch(isMobile, (m) => {
  statsActive.value = m ? [] : ['stats']
})

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

// 主要食材（非必须通过条目 Note 备注表达，不再有独立 optional 分组）
const mainIngredients = computed(() => activeVersion.value.ingredients.main || [])

function resolveAsset(p: string): string {
  if (/^(https?:)?\/\//.test(p)) return p
  return `/assets/${p}`
}
</script>

<style scoped>
.recipe-detail {
  border-radius: 12px;
}

.detail-title {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 6px;
}

.detail-head-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.shot-btn {
  flex-shrink: 0;
}

.detail-head-top .shot-btn .el-icon + span {
  margin-left: 4px;
}

.detail-tags .tag {
  margin-right: 8px;
  margin-bottom: 2px;
}

.detail-desc {
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  line-height: 1.55;
}

.version-tabs {
  margin-top: 4px;
}

/* 双栏布局：主列（步骤优先）+ 侧栏（统计/主料/调料） */
.detail-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  margin-top: 12px;
}

.detail-main {
  flex: 1 1 62%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.detail-side {
  flex: 0 0 34%;
  min-width: 0;
  position: sticky;
  top: 76px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.block-title {
  font-weight: 600;
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

/* 统计卡（紧凑） */
.stats-card {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  text-align: center;
}

.stat-icon {
  font-size: 18px;
}

.stat-label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.stat-value {
  font-size: 13px;
  font-weight: 600;
}

/* 配菜/非必须折叠 */
.minor-collapse {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

.minor-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
}

/* 统计折叠（时间/难度） */
.stats-collapse {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

.stats-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
}

.stats-card {
  border: none;
  border-radius: 0;
  background: transparent;
  padding: 4px 8px 8px;
}

.ing-group {
  margin-bottom: 12px;
}

.ing-group-title {
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}

.step-desc {
  line-height: 1.55;
  margin-bottom: 6px;
  white-space: pre-wrap;
}

/* 步骤图（桌面较大） */
.step-img {
  width: 210px;
  height: 130px;
  border-radius: 8px;
}

.images {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.gallery-img {
  width: 160px;
  height: 108px;
  border-radius: 8px;
}

.video {
  margin-top: 8px;
}

/* H5 自适应：单列 + 步骤优先（display:contents 让各块按 order 重排） */
@media (max-width: 768px) {
  .recipe-detail :deep(.el-card__body) {
    padding: 12px !important;
  }
  .detail-title {
    font-size: 21px;
  }
  .detail-desc {
    font-size: 13px;
  }

  .detail-layout {
    flex-direction: column;
    gap: 12px;
  }
  /* 移除 main/side 容器盒，块成为 layout 直接子项以应用 order */
  .detail-main,
  .detail-side {
    display: contents;
  }

  .order-1 { order: 1; } /* 主要食材（含可选） */
  .order-2 { order: 2; } /* 配菜/辅料 */
  .order-3 { order: 3; } /* 调料 */
  .order-4 { order: 4; } /* 步骤 */
  .order-5 { order: 5; } /* 统计（时间/难度，折叠藏起） */
  .order-6 { order: 6; } /* 过程图/视频 */

  /* 统计卡 2x2 */
  .stats-card {
    grid-template-columns: repeat(2, 1fr);
  }

  /* 步骤图移动端全宽 16:9 */
  .step-img {
    width: 100%;
    height: auto;
    aspect-ratio: 16 / 9;
  }

  .step-desc {
    font-size: 13px;
  }

  .gallery-img {
    width: 100%;
    height: auto;
    aspect-ratio: 16 / 9;
  }

  /* 食材/调料表格在窄屏可横向滚动，避免溢出 */
  .detail-main :deep(.el-table),
  .detail-side :deep(.el-table) {
    width: 100%;
    overflow-x: auto;
    display: block;
  }
}
</style>
