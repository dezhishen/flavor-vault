import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'node:fs'
import path from 'node:path'

// 使用相对路径 base，兼容 GitHub Pages 子路径部署
export default defineConfig({
  plugins: [
    vue(),
    {
      // 开发服务器：将 /data/* 代理到项目根 dist/data
      name: 'serve-vault-data',
      configureServer(server) {
        const distData = path.resolve(__dirname, '../dist/data')
        server.middlewares.use('/data', (req, res, next) => {
          const urlPath = (req.url || '').split('?')[0]
          const file = path.join(distData, urlPath)
          if (file.startsWith(distData) && fs.existsSync(file) && fs.statSync(file).isFile()) {
            res.setHeader('Content-Type', 'application/json; charset=utf-8')
            fs.createReadStream(file).pipe(res)
            return
          }
          next()
        })
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

