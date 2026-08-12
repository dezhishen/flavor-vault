<template>
  <el-container class="app">
    <el-header class="app-header">
      <router-link to="/" class="brand">
        <span class="logo">🍲</span>
        <span class="title">Flavor Vault</span>
      </router-link>
      <span class="subtitle">· 菜谱库</span>
    </el-header>

    <el-main class="app-main">
      <router-view />
    </el-main>

    <el-footer class="app-footer">
      Flavor Vault · 菜谱库
    </el-footer>
  </el-container>
</template>

<script setup lang="ts">
// 根组件：负责整体布局
</script>

<style scoped>
.app {
  min-height: 100vh;
  background: var(--el-bg-color-page);
}

.app-header {
  display: flex;
  align-items: center;
  gap: 8px;
  background: linear-gradient(135deg, #f56c6c 0%, #e6a23c 100%);
  color: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: #fff;
  font-size: 20px;
  font-weight: 700;
}

.logo {
  font-size: 26px;
}

.subtitle {
  opacity: 0.85;
  font-size: 14px;
}

.app-main {
  max-width: 1200px;
  margin: 0 auto;
  width: 100%;
  padding: 16px;
}

.app-footer {
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 60px;
}

/* H5 自适应 */
@media (max-width: 768px) {
  .app-header {
    height: 52px;
    padding: 0 12px;
    gap: 4px;
  }
  .brand {
    font-size: 17px;
  }
  .logo {
    font-size: 22px;
  }
  .subtitle {
    display: none;
  }
  .app-main {
    padding: 12px;
  }
  .app-footer {
    font-size: 11px;
    line-height: 48px;
  }
}
</style>
