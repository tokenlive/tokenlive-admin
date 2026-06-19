<template>
    <div
        ref="chartRef"
        class="x-chart"></div>
</template>

<script setup>
import * as echarts from 'echarts'
import { markRaw, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAppStore } from '@/store'
import { storeToRefs } from 'pinia'

defineOptions({
    name: 'XChart',
})

/**
 * 图表
 * @property {object} options 配置信息
 * @property {number | string} width 宽
 * @property {number | string} height 高
 * @property {boolean} loading 是否处于加载中状态
 */
const props = defineProps({
    options: {
        type: Object,
        default: () => ({}),
    },
    width: {
        type: [Number, String],
        default: 'auto',
    },
    height: {
        type: [Number, String],
        default: 'auto',
    },
    loading: {
        type: Boolean,
        default: false,
    },
})

const emit = defineEmits(['init'])

const chart = ref(null)
const chartRef = ref()
const appStore = useAppStore()
const { config } = storeToRefs(appStore)

let resizeObserver = null

// 监听主题变化，重建图表实例以适用新主题的视觉配色
watch(
    () => config.value.theme,
    () => {
        if (chart.value) {
            chart.value.dispose()
            chart.value = null
        }
        init()
    }
)

// 深度监听 options 属性，只有当实例已经创建时，使用 setOption 更新数据，激活过渡动画
watch(
    () => props.options,
    (newVal) => {
        if (chart.value) {
            chart.value.setOption(
                {
                    backgroundColor: 'transparent',
                    ...newVal,
                },
                true
            )
        }
    },
    {
        deep: true,
    }
)

// 监听 loading 属性，从而展示 ECharts 的 Loading 图层
watch(
    () => props.loading,
    (val) => {
        if (chart.value) {
            if (val) {
                showChartLoading()
            } else {
                chart.value.hideLoading()
            }
        }
    }
)

onMounted(() => {
    init()

    // 启用 ResizeObserver 精准监听容器缩放
    if (window.ResizeObserver && chartRef.value) {
        resizeObserver = new ResizeObserver(() => {
            if (chart.value) {
                chart.value.resize()
            }
        })
        resizeObserver.observe(chartRef.value)
    } else {
        window.addEventListener('resize', handleResize)
    }
})

function handleResize() {
    if (chart.value) {
        chart.value.resize()
    }
}

onBeforeUnmount(() => {
    if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
    } else {
        window.removeEventListener('resize', handleResize)
    }

    if (chart.value) {
        chart.value.dispose()
        chart.value = null
    }
})

// 显示 ECharts 加载动效
function showChartLoading() {
    if (!chart.value) return
    const isDark = config.value.theme === 'dark'
    chart.value.showLoading({
        text: '',
        color: 'var(--color-primary, #7c5cfc)',
        textColor: isDark ? '#fff' : '#000',
        maskColor: isDark ? 'rgba(20, 20, 20, 0.4)' : 'rgba(255, 255, 255, 0.4)',
        zlevel: 0,
    })
}

/**
 * 初始化
 * @private
 */
function init() {
    if (chart.value) return

    echarts.registerTheme('chart', {
        legend: {
            itemWidth: 14,
            itemHeight: 14,
        },
        bar: {
            barWidth: 30,
            backgroundStyle: {
                color: 'rgba(180, 180, 180, 0.2)',
            },
            showBackground: true,
        },
    })

    const instance = echarts.init(chartRef.value, config.value.theme === 'dark' ? 'dark' : 'chart', {
        width: props.width,
        height: props.height,
    })
    chart.value = markRaw(instance)

    if (props.loading) {
        showChartLoading()
    }

    setTimeout(() => {
        if (chart.value) {
            chart.value.setOption(
                {
                    backgroundColor: 'transparent',
                    ...props.options,
                },
                true
            )
            chart.value.resize()
            if (!props.loading) {
                chart.value.hideLoading()
            }
            emit('init', chart.value)
        }
    }, 100)
}
</script>

<style lang="less" scoped>
.x-chart {
    width: 100%;
    height: 100%;
}
</style>
