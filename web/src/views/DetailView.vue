<template>
  <div class="detail-page">
    <el-page-header @back="router.back()" class="detail-header">
      <template #content>
        <span class="detail-title">{{ detail?.name ?? '菜谱详情' }}</span>
      </template>
    </el-page-header>

    <div v-if="loading" class="skeleton">
      <el-skeleton :rows="6" animated />
    </div>

    <el-empty v-else-if="!detail" description="未找到该菜谱">
      <el-button type="primary" @click="router.push('/')">返回首页</el-button>
    </el-empty>

    <RecipeDetail v-else :detail="detail" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import RecipeDetail from '../components/RecipeDetail.vue'
import { useRecipeStore } from '../stores/recipe'
import type { RecipeDetail as RecipeDetailType } from '../types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const recipeStore = useRecipeStore()

const detail = ref<RecipeDetailType | null>(null)
const loading = ref(false)

async function fetchDetail() {
  loading.value = true
  detail.value = await recipeStore.loadDetail(props.id)
  loading.value = false
}

onMounted(fetchDetail)
watch(() => props.id, fetchDetail)
</script>

<style scoped>
.detail-page {
  max-width: 860px;
  margin: 0 auto;
}

.detail-header {
  margin-bottom: 20px;
}

.detail-title {
  font-weight: 600;
}

.skeleton {
  padding: 20px 0;
}
</style>
