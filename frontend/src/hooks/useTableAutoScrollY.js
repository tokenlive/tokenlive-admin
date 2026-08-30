import { computed, onActivated, onBeforeUnmount, onDeactivated, ref, watch } from 'vue'

/**
 * 表格高度自适应
 *
 * 监听表格区域容器的尺寸变化，动态计算 a-table 的 scroll.y：
 * 容器高度 - 表头高度 - 分页高度（含外边距），并暴露同一数值供 CSS 变量
 * 将表体 min-height 撑满，保证数据较少时分页仍贴底。
 *
 * 用法（配合 app-layout.less 中的 .app-card--fill / .table-fill-region）：
 *   const { scrollY, containerRef, containerStyle } = useTableAutoScrollY()
 *   <div ref="containerRef" class="table-fill-region" :style="containerStyle">
 *       <a-table :scroll="{ x: 'max-content', y: scrollY || undefined }" />
 *   </div>
 *
 * @param {object} options
 * @param {number} options.minHeight 计算结果的最小值下限，防止极小窗口下表体塌缩
 */
export default (options = {}) => {
    const { minHeight = 96 } = options

    const containerRef = ref(null)
    const scrollY = ref(0)

    const containerStyle = computed(() => ({
        '--auto-table-body-min-height': `${scrollY.value}px`,
    }))

    /**
     * 元素高度含垂直外边距
     */
    function outerHeight(el) {
        if (!el) return 0
        const style = window.getComputedStyle(el)
        return el.offsetHeight + parseFloat(style.marginTop || 0) + parseFloat(style.marginBottom || 0)
    }

    function measure() {
        const container = containerRef.value
        if (!container || !container.clientHeight) {
            // keep-alive 切走等场景容器不可见时保留上次结果
            return
        }
        const thead = container.querySelector('.ant-table-thead')
        const pagination = container.querySelector('.ant-pagination')
        if (!thead) {
            // 表格尚未渲染时不限高，保持默认行为
            scrollY.value = 0
            return
        }
        const available = container.clientHeight - thead.offsetHeight - outerHeight(pagination)
        scrollY.value = Math.max(minHeight, Math.floor(available))
    }

    let resizeObserver = null
    let mutationObserver = null
    let frameId = 0

    // rAF 合并同帧内的多次触发
    function scheduleMeasure() {
        if (frameId) return
        frameId = requestAnimationFrame(() => {
            frameId = 0
            measure()
        })
    }

    function unbind() {
        resizeObserver?.disconnect()
        mutationObserver?.disconnect()
        resizeObserver = null
        mutationObserver = null
    }

    function bind(el) {
        unbind()
        if (!el) return
        resizeObserver = new ResizeObserver(scheduleMeasure)
        resizeObserver.observe(el)
        // 分页/表头的挂载与卸载不改变容器自身尺寸，用 MutationObserver 补充感知
        mutationObserver = new MutationObserver(scheduleMeasure)
        mutationObserver.observe(el, { childList: true, subtree: true })
        scheduleMeasure()
    }

    watch(containerRef, (el) => bind(el), { flush: 'post', immediate: true })

    onActivated(() => {
        bind(containerRef.value)
    })

    onDeactivated(() => {
        if (frameId) {
            cancelAnimationFrame(frameId)
            frameId = 0
        }
    })

    onBeforeUnmount(() => {
        unbind()
        if (frameId) cancelAnimationFrame(frameId)
    })

    return {
        scrollY,
        containerRef,
        containerStyle,
    }
}
