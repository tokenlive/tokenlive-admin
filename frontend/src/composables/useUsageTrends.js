import { onActivated, onDeactivated, onMounted, onUnmounted, reactive, watch } from 'vue'
import { getUsageTrends } from '@/apis/modules/usage-trends'
import { createUsageTrendsController } from '@/utils/usage-trends-state'

export function useUsageTrends(authorized) {
    const state = reactive({})
    const controller = createUsageTrendsController({
        load: getUsageTrends,
        onChange: (next) => Object.assign(state, next),
    })
    Object.assign(state, controller.getState())
    watch(authorized, (value) => controller.setAuthorized(value), { immediate: true, flush: 'sync' })
    const syncVisibility = () => controller.setVisible(document.visibilityState === 'visible')
    onMounted(() => {
        document.addEventListener('visibilitychange', syncVisibility)
        syncVisibility()
        controller.setActive(true)
    })
    onActivated(() => {
        syncVisibility()
        controller.setActive(true)
    })
    onDeactivated(() => controller.setActive(false))
    onUnmounted(() => {
        document.removeEventListener('visibilitychange', syncVisibility)
        controller.dispose()
    })
    return { state, refresh: controller.refresh, setTimeRange: controller.setTimeRange }
}
