import { computed, onActivated, onDeactivated, onMounted, onUnmounted, reactive, watch } from 'vue'
import { getAPIKeyRanking } from '@/apis/modules/api-key-usage'
import { createAPIKeyUsageController } from '@/utils/api-key-usage-state'

export function useAPIKeyUsage(authorized) {
    const state = reactive({})
    const controller = createAPIKeyUsageController({
        load: getAPIKeyRanking,
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
    return {
        state,
        query: computed(() => state.query),
        setQuery: controller.setQuery,
        refresh: controller.refresh,
    }
}
