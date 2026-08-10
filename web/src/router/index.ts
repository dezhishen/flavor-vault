import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import DetailView from '../views/DetailView.vue'

// 使用 hash 路由，静态托管（GitHub Pages）无需服务器重写规则
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/recipe/:id', name: 'detail', component: DetailView, props: true },
  ],
})

export default router
