import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { FilterSelections } from '../types'

/**
 * filterStore：管理当前选中的厨具/标签/食材过滤条件。
 * 前端在内存中基于 filters.json 倒排索引求交集。
 */
export const useFilterStore = defineStore('filter', () => {
  const kitchenware = ref<string[]>([])
  const tags = ref<string[]>([])
  const ingredients = ref<string[]>([])
  /** 搜索关键词（来自搜索框，与筛选条件取交集） */
  const keyword = ref('')

  const selections = computed<FilterSelections>(() => ({
    kitchenware: kitchenware.value,
    tags: tags.value,
    ingredients: ingredients.value,
  }))

  const isActive = computed(
    () =>
      kitchenware.value.length > 0 ||
      tags.value.length > 0 ||
      ingredients.value.length > 0 ||
      keyword.value.trim() !== '',
  )

  function reset() {
    kitchenware.value = []
    tags.value = []
    ingredients.value = []
    keyword.value = ''
  }

  return { kitchenware, tags, ingredients, keyword, selections, isActive, reset }
})
