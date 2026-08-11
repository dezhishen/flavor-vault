<template>
  <div class="recipe-list">
    <div class="list-toolbar">
      <span v-if="recipeStore.loading" class="hint">正在加载数据...</span>
      <span v-else-if="recipeStore.error" class="hint error">
        {{ recipeStore.error }}
      </span>
      <span v-else class="count">
        共
        <b>{{ filteredRecipes.length }}</b>
        / {{ recipeStore.summaries.length }} 道菜谱
      </span>
    </div>

    <div v-if="recipeStore.loading" class="grid">
      <el-card v-for="i in 6" :key="i" shadow="hover" class="recipe-card">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="image" style="width: 100%; height: 140px" />
            <el-skeleton-item variant="h3" style="width: 50%" />
            <el-skeleton-item variant="text" style="width: 90%" />
          </template>
        </el-skeleton>
      </el-card>
    </div>

    <el-empty
      v-else-if="filteredRecipes.length === 0"
      description="没有符合条件的菜谱"
    />

    <div v-else class="grid">
      <RecipeCard
        v-for="r in filteredRecipes"
        :key="r.id"
        :recipe="r"
        @click="router.push(`/recipe/${r.id}`)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import RecipeCard from './RecipeCard.vue'
import { useFilter } from '../composables/useFilter'
import { useRecipe } from '../composables/useRecipe'

const router = useRouter()
const { filteredRecipes } = useFilter()
const { recipeStore } = useRecipe()
</script>

<style scoped>
.list-toolbar {
  margin-bottom: 16px;
}

.count {
  color: var(--el-text-color-regular);
  font-size: 14px;
}

.hint {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.hint.error {
  color: var(--el-color-danger);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

/* H5 自适应：小屏 2 列，超窄 1 列 */
@media (max-width: 700px) {
  .grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }
}
@media (max-width: 380px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
