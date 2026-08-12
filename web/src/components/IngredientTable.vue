<template>
  <el-table :data="items" size="small" border>
    <el-table-column prop="name" :label="nameLabel" min-width="90" />
    <el-table-column prop="amount" label="用量" min-width="80" />
    <!-- 可替换并入备注列内另起一行，避免独立列被盖住 -->
    <el-table-column label="备注 / 可替换" min-width="140">
      <template #default="{ row }">
        <div class="note-cell">
          <span v-if="row.note" class="note-text">{{ row.note }}</span>
          <div v-if="(row.alternatives || []).length" class="alt-line">
            <el-tag
              v-for="(a, j) in row.alternatives || []"
              :key="j"
              size="small"
              type="success"
              effect="plain"
              class="alt-tag"
            >
              可换 {{ a.name }}{{ a.amount ? ` ${a.amount}` : '' }}{{ a.note ? `（${a.note}）` : '' }}
            </el-tag>
          </div>
        </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
interface AltItem {
  name: string
  amount?: string
  note?: string
}
interface Item {
  name: string
  amount?: string
  note?: string
  alternatives?: AltItem[]
}
withDefaults(
  defineProps<{ items: Item[]; nameLabel?: string }>(),
  { nameLabel: '食材' }
)
</script>

<style scoped>
.note-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.note-text {
  white-space: normal;
}

.alt-line {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.alt-tag {
  margin: 0;
}
</style>
