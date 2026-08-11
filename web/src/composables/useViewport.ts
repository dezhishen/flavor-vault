import { onMounted, onUnmounted, ref } from 'vue'

const MOBILE_WIDTH = 768

/**
 * useViewport：响应式判断是否为移动端（宽度 < 768px）。
 * 用于需要根据屏幕切换布局/列数的场景。
 */
export function useViewport() {
  const width = ref(typeof window !== 'undefined' ? window.innerWidth : 1200)
  const isMobile = ref(width.value < MOBILE_WIDTH)

  function onResize() {
    width.value = window.innerWidth
    isMobile.value = width.value < MOBILE_WIDTH
  }

  onMounted(() => window.addEventListener('resize', onResize))
  onUnmounted(() => window.removeEventListener('resize', onResize))

  return { width, isMobile }
}
