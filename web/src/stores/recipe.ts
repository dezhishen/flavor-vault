import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { FacetIndex, Meta, RecipeDetail, RecipeSummary, SearchEntry } from '../types'

/** 数据 API 基础路径（站点部署在域名根路径 /data；Vite dev 代理到构建产物） */
const DATA_BASE = '/data'

/**
 * recipeStore：启动时加载 meta.json、all.json、filters.json 与 search.json，
 * 懒加载 details/{id}.json。
 */
export const useRecipeStore = defineStore('recipe', () => {
  const meta = ref<Meta | null>(null)
  const summaries = ref<RecipeSummary[]>([])
  const facet = ref<FacetIndex | null>(null)
  const searchIndex = ref<SearchEntry[]>([])
  const loaded = ref(false)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const detailCache = ref<Record<string, RecipeDetail>>({})

  async function loadAll() {
    if (loaded.value) return
    loading.value = true
    error.value = null
    try {
      const [metaRes, allRes, facetRes, searchRes] = await Promise.all([
        fetch(`${DATA_BASE}/meta.json`),
        fetch(`${DATA_BASE}/all.json`),
        fetch(`${DATA_BASE}/filters.json`),
        fetch(`${DATA_BASE}/search.json`),
      ])
      if (!metaRes.ok || !allRes.ok || !facetRes.ok || !searchRes.ok) {
        throw new Error('数据加载失败，请稍后再试')
      }
      meta.value = await metaRes.json()
      summaries.value = await allRes.json()
      facet.value = await facetRes.json()
      searchIndex.value = await searchRes.json()
      loaded.value = true
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  async function loadDetail(id: string): Promise<RecipeDetail | null> {
    if (detailCache.value[id]) {
      return detailCache.value[id]
    }
    try {
      const res = await fetch(`${DATA_BASE}/details/${id}.json`)
      if (!res.ok) return null
      const detail: RecipeDetail = await res.json()
      detailCache.value = { ...detailCache.value, [id]: detail }
      return detail
    } catch {
      return null
    }
  }

  const byId = computed(() => {
    const map = new Map<string, RecipeSummary>()
    for (const s of summaries.value) map.set(s.id, s)
    return map
  })

  return {
    meta,
    summaries,
    facet,
    searchIndex,
    loaded,
    loading,
    error,
    byId,
    loadAll,
    loadDetail,
  }
})
