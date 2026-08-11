import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'node:fs'
import path from 'node:path'

// 使用相对路径 base，兼容 GitHub Pages 子路径部署
export default defineConfig({
  plugins: [
    vue(),
    {
      // 开发服务器：把 /data/* 与 /assets/* 代理到项目根 dist（本地润 dev 的数据与图片）
      name: 'serve-vault-dist',
      configureServer(server) {
        const distRoot = path.resolve(__dirname, '../dist')
        const serve = (base: string, mime: string) => {
          server.middlewares.use(base, (req, res, next) => {
            const urlPath = (req.url || '').split('?')[0]
            const file = path.join(distRoot, base, urlPath)
            if (file.startsWith(distRoot) && fs.existsSync(file) && fs.statSync(file).isFile()) {
              res.setHeader('Content-Type', mime)
              fs.createReadStream(file).pipe(res)
              return
            }
            next()
          })
        }
        serve('/data', 'application/json; charset=utf-8')
        serve('/assets', 'application/octet-stream')
      },
    },
  ],
  base: './',
  build: {
    outDir: 'dist',
    chunkSizeWarningLimit: 1500,
  },
  server: {
    port: 5173,
  },
})

