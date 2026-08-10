<template>
  <div class="home">
    <el-alert
      v-if="recipeStore.error"
      :title="recipeStore.error"
      type="error"
      show-icon
      :closable="false"
      class="load-error"
    />
    <FilterPanel v-if="recipeStore.loaded" />
    <RecipeList />
  </div>
</template>

<script setup lang="ts">
import FilterPanel from '../components/FilterPanel.vue'
import RecipeList from '../components/RecipeList.vue'
import { useRecipeStore } from '../stores/recipe'

const recipeStore = useRecipeStore()
</script>

<style scoped>
.load-error {
  margin-bottom: 16px;
}
</style>
