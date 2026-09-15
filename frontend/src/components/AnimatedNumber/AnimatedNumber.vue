<template>
    <span class="animated-number">{{ displayValue }}</span>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'

defineOptions({
    name: 'AnimatedNumber',
})

const props = defineProps({
    /** 目标数值 */
    value: {
        type: [Number, String],
        default: 0,
    },
    /** 动画持续时长（毫秒） */
    duration: {
        type: Number,
        default: 800,
    },
    /** 小数位数精度 */
    precision: {
        type: Number,
        default: 0,
    },
    /** 是否显示千分位逗号 */
    separator: {
        type: Boolean,
        default: true,
    },
    /** 前缀 */
    prefix: {
        type: String,
        default: '',
    },
    /** 后缀 */
    suffix: {
        type: String,
        default: '',
    },
    /** 自定义格式化函数：(val: number) => string */
    formatter: {
        type: Function,
        default: null,
    },
})

const displayValue = ref('')

let currentNum = 0
let startNum = 0
let targetNum = 0
let startTime = 0
let rafId = null

function parseNumber(val) {
    if (val === null || val === undefined || val === '') return 0
    const num = Number(val)
    return Number.isFinite(num) ? num : 0
}

// 缓动算法：Cubic Ease-Out，先快后慢，丝滑入轨
function easeOutCubic(x) {
    return 1 - Math.pow(1 - x, 3)
}

function formatOutput(num) {
    if (props.formatter) {
        return props.formatter(num)
    }

    const fixed = num.toFixed(Math.max(0, props.precision))
    const parts = fixed.split('.')
    let integerPart = parts[0]
    const decimalPart = parts[1]

    if (props.separator) {
        integerPart = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    }

    const formatted = decimalPart !== undefined ? `${integerPart}.${decimalPart}` : integerPart
    return `${props.prefix}${formatted}${props.suffix}`
}

function updateDisplay(num) {
    displayValue.value = formatOutput(num)
}

function tick(currentTime) {
    if (!startTime) startTime = currentTime
    const elapsed = currentTime - startTime
    const progress = Math.min(elapsed / Math.max(1, props.duration), 1)
    const easedProgress = easeOutCubic(progress)

    currentNum = startNum + (targetNum - startNum) * easedProgress
    updateDisplay(currentNum)

    if (progress < 1) {
        rafId = requestAnimationFrame(tick)
    } else {
        currentNum = targetNum
        updateDisplay(currentNum)
        rafId = null
    }
}

function startAnimation(newTarget) {
    if (rafId) {
        cancelAnimationFrame(rafId)
        rafId = null
    }

    targetNum = parseNumber(newTarget)
    startNum = currentNum

    // 如果起止数值相同或无动画时间，直接呈现
    if (startNum === targetNum || props.duration <= 0) {
        currentNum = targetNum
        updateDisplay(currentNum)
        return
    }

    startTime = performance.now()
    rafId = requestAnimationFrame(tick)
}

// 监听目标数值变化，支持连续推送打断平滑变轨
watch(
    () => props.value,
    (newVal) => {
        startAnimation(newVal)
    }
)

// 页面可见性监听：后台休眠时避免累积无意义的 RAF 计算
function handleVisibilityChange() {
    if (document.hidden) {
        if (rafId) {
            cancelAnimationFrame(rafId)
            rafId = null
        }
        currentNum = targetNum
        updateDisplay(currentNum)
    }
}

onMounted(() => {
    document.addEventListener('visibilitychange', handleVisibilityChange)
    const initialTarget = parseNumber(props.value)
    // 首次挂载时从 0 平滑滚动到目标值
    currentNum = 0
    startAnimation(initialTarget)
})

onUnmounted(() => {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    if (rafId) {
        cancelAnimationFrame(rafId)
        rafId = null
    }
})
</script>

<style scoped>
.animated-number {
    display: inline-block;
    font-variant-numeric: tabular-nums;
}
</style>
