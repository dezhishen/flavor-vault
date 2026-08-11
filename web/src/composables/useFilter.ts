import { computed } from 'vue'
import { useRecipeStore } from '../stores/recipe'
import { useFilterStore } from '../stores/filter'
import type { RecipeSummary } from '../types'

/**
 * useFilter：基于倒排索引在内存中求交集（筛选），
 * 并与搜索关键词（search.json 全文匹配）取交集，
 * 响应筛选/搜索条件变化自动更新结果列表。
 */
export function useFilter() {
  const recipeStore = useRecipeStore()
  const filterStore = useFilterStore()

  /** 匹配的 ID 集合（倒排索引交集） */
  const matchedIds = computed(() => {
    const facet = recipeStore.facet
    if (!facet) return new Set<string>()

    const lists: string[][] = []
    for (const kw of filterStore.kitchenware) {
      lists.push(facet.kitchenware[kw] ?? [])
    }
    for (const t of filterStore.tags) {
      lists.push(facet.tags[t] ?? [])
    }
    for (const ing of filterStore.ingredients) {
      lists.push(facet.ingredients[ing] ?? [])
    }
    if (lists.length === 0) {
      return new Set(recipeStore.summaries.map((s) => s.id))
    }
    return new Set(intersect(lists))
  })

  /** 关键词命中的 ID 集合（null 表示无关键词，不限制） */
  const keywordIds = computed<Set<string> | null>(() => {
    const kw = filterStore.keyword.trim().toLowerCase()
    if (!kw) return null
    const tokens = kw.split(/\s+/).filter(Boolean)
    if (tokens.length === 0) return null
    const index = recipeStore.searchIndex
    const matched = new Set<string>()
    for (const e of index) {
      const hay = [
        e.name,
        e.description,
        ...e.tags,
        ...e.kitchenware,
        ...e.ingredients,
        e.steps,
      ]
        .join(' ')
        .toLowerCase()
      if (tokens.every((t) => hay.includes(t))) matched.add(e.id)
    }
    return matched
  })

  /** 过滤后的菜谱列表（保持 all.json 顺序） */
  const filteredRecipes = computed<RecipeSummary[]>(() =>
    recipeStore.summaries.filter((s) => {
      if (!matchedIds.value.has(s.id)) return false
      if (keywordIds.value && !keywordIds.value.has(s.id)) return false
      return true
    }),
  )

  /** 可用的筛选选项（来自倒排索引 key） */
  const options = computed(() => {
    const facet = recipeStore.facet
    if (!facet) return { kitchenware: [], tags: [], ingredients: [] }
    return {
      kitchenware: Object.keys(facet.kitchenware).sort(),
      tags: Object.keys(facet.tags).sort(),
      ingredients: Object.keys(facet.ingredients).sort(),
    }
  })

  return { matchedIds, filteredRecipes, options }
}

/** 多列表交集（各列表视为已排序） */
export function intersect(lists: string[][]): string[] {
  if (lists.length === 0) return []
  let result = [...lists[0]]
  for (let i = 1; i < lists.length; i++) {
    const set = new Set(lists[i])
    result = result.filter((x) => set.has(x))
    if (result.length === 0) break
  }
  return result
}
