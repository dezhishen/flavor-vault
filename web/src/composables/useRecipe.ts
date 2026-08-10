import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRecipeStore } from '../stores/recipe'

/**
 * useRecipe：数据加载入口。
 * 组件挂载时触发启动加载（meta + all + filters）。
 */
export function useRecipe() {
  const recipeStore = useRecipeStore()
  const { meta, summaries, facet, loaded, loading, error, byId } =
    storeToRefs(recipeStore)

  onMounted(() => {
    recipeStore.loadAll()
  })

  return { meta, summaries, facet, loaded, loading, error, byId, recipeStore }
}
