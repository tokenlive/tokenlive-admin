import { onUnmounted, ref } from 'vue'
import { getUpgradeCapability, getUpgradeTask, prepareUpgrade, submitUpgrade } from '@/apis/modules/system'
import { createUpgradeController } from '@/utils/upgrade-state'

// Owns the one upgrade controller at layout level. Authorization follows the
// summary's can_manage_upgrades flag; activity follows the About dialog.
// onSucceeded fires after a confirmed upgrade finished so the layout can
// reload version data.
export function useUpgrade(onSucceeded = () => {}) {
    const state = ref(null)
    const controller = createUpgradeController({
        api: {
            capability: getUpgradeCapability,
            prepare: prepareUpgrade,
            submit: submitUpgrade,
            task: getUpgradeTask,
        },
        onChange: (next) => {
            state.value = next
        },
        onSucceeded,
    })
    onUnmounted(() => controller.dispose())

    function sync(summaryValue, aboutOpen) {
        controller.setAuthorized(summaryValue?.can_manage_upgrades === true)
        controller.setActive(aboutOpen === true)
    }
    return { state, controller, sync }
}
