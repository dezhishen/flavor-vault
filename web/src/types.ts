// 与 Go 端生成的数据结构保持一致

/** 轻量菜谱摘要（all.json / by-tag/*.json） */
export interface RecipeSummary {
  id: string
  name: string
  description: string
  tags: string[]
  kitchenware: string[]
  ingredients: string[]
  cover: string
  prep_time: number
  cook_time: number
  difficulty: number
}

export interface Ingredient {
  name: string
  amount: string
}

export interface Step {
  order: number
  description: string
  image_ref?: string
}

export interface Media {
  cover: string
  images: string[]
  video_url?: string
}

export interface Stats {
  prep_time: number
  cook_time: number
  difficulty: number
}

/** 完整菜谱详情（details/{id}.json） */
export interface RecipeDetail {
  id: string
  name: string
  description: string
  tags: string[]
  kitchenware: string[]
  ingredients: {
    main: Ingredient[]
    side: Ingredient[]
  }
  steps: Step[]
  media: Media
  stats: Stats
  created_at: string
  updated_at: string
}

/** 倒排索引（filters.json） */
export interface FacetIndex {
  kitchenware: Record<string, string[]>
  ingredients: Record<string, string[]>
  tags: Record<string, string[]>
}

/** 统计信息（meta.json） */
export interface Meta {
  total: number
  generated_at: string
  tags: Record<string, number>
  kitchenware: Record<string, number>
  difficulty: Record<string, number>
  avg_prep_time: number
  avg_cook_time: number
  avg_total_time: number
  all_count: number
}

/** 过滤条件 */
export interface FilterSelections {
  kitchenware: string[]
  tags: string[]
  ingredients: string[]
}
