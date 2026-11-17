const STORAGE_PREFIX = 'tokenlive.dashboard.panelDisabled.'

function getStorage() {
    try {
        if (typeof window === 'undefined' || !window.localStorage) return null
        return window.localStorage
    } catch {
        return null
    }
}

/**
 * 读取面板上次记录的禁用状态，仅作为首屏是否预渲染骨架占位的提示。
 * @param {string} panelKey
 * @returns {boolean}
 */
export function isPanelRememberedDisabled(panelKey) {
    const storage = getStorage()
    if (!storage) return false
    try {
        return storage.getItem(`${STORAGE_PREFIX}${panelKey}`) === '1'
    } catch {
        return false
    }
}

/**
 * 记忆或清除面板的禁用状态（实时接口返回后始终以接口为准）。
 * @param {string} panelKey
 * @param {boolean} disabled
 */
export function rememberPanelDisabled(panelKey, disabled) {
    const storage = getStorage()
    if (!storage) return
    try {
        const key = `${STORAGE_PREFIX}${panelKey}`
        if (disabled) storage.setItem(key, '1')
        else storage.removeItem(key)
    } catch {
        // 忽略隐私模式或存储配额异常
    }
}
