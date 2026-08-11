import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import DetailView from '../views/DetailView.vue'

// 使用 history 路由（URL 无 #）：站点部署在自定义域名根路径，配合 404.html 实现深链刷新回退
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/recipe/:id', name: 'detail', component: DetailView, props: true },
  ],
})

export default router
