<template>
  <div v-if="!isMobile" class="qr-fab">
    <el-popover placement="top-end" trigger="click" :width="200" popper-class="qr-pop">
      <template #reference>
        <div class="qr-fab-btn" title="手机扫码打开">
          <span class="qr-fab-icon">📱</span>
        </div>
      </template>
      <div class="qr-card">
        <img v-if="qr" :src="qr" alt="二维码" class="qr-img" />
        <div v-else class="qr-img qr-loading">生成中…</div>
        <div class="qr-tip">手机扫码打开当前菜谱</div>
        <div class="qr-url">{{ url }}</div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import { useViewport } from '../composables/useViewport'

const { isMobile } = useViewport()
const qr = ref('')
const url = ref('')

onMounted(async () => {
  url.value = window.location.href
  try {
    qr.value = await QRCode.toDataURL(url.value, {
      width: 180,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
  } catch (e) {
    console.error('生成二维码失败', e)
  }
})
</script>

<style scoped>
.qr-fab {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 100;
}

.qr-fab-btn {
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
  transition: transform 0.15s ease;
}

.qr-fab-btn:hover {
  transform: scale(1.08);
}

.qr-fab-icon {
  font-size: 22px;
  line-height: 1;
}

.qr-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
}

.qr-img {
  width: 180px;
  height: 180px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.qr-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.qr-tip {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.qr-url {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  max-width: 180px;
  line-height: 1.4;
}
</style>
