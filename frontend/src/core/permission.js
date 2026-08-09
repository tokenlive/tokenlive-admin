import { createProgress } from '@/plugins/progress'
import router from '@/router'
import { whiteList } from '@/router/config'
import { useAppStore, useRouterStore, useUserStore } from '@/store'
import { getStartupAction } from '@/utils/session'

const progress = createProgress()

function loginRoute() {
    return {
        name: 'login',
        replace: true,
        query: {
            redirect: encodeURIComponent(location.href),
        },
    }
}

async function restoreSession(userStore) {
    const action = getStartupAction({
        hasAccessToken: userStore.isLogin,
        hasRefreshToken: userStore.hasRefreshToken,
    })

    if (action === 'login') return false
    if (action === 'refresh') return userStore.refreshAccessToken()
    return true
}

async function initializeApp(appStore, userStore) {
    const restored = await restoreSession(userStore)
    if (!restored) return { restored: false, initialized: false }

    const initialized = !appStore.complete
    if (initialized) await appStore.init()
    return { restored: true, initialized }
}

router.beforeEach(async (to, from, next) => {
    const { meta } = to
    const { title } = meta
    const appStore = useAppStore()
    const userStore = useUserStore()

    progress.start()
    document.title = title ? `${title} - ${import.meta.env.VITE_TITLE}` : import.meta.env.VITE_TITLE

    if (to.name === 'login') {
        try {
            const { restored } = await initializeApp(appStore, userStore)
            if (!restored) {
                next()
                return
            }
            const routerStore = useRouterStore()
            next(routerStore.indexRoute || { name: 'index' })
        } catch (error) {
            // 临时网络或服务错误只展示登录页，保留 token 供下次恢复
            next()
        }
        return
    }

    if (whiteList.includes(to.name)) {
        next()
        return
    }

    try {
        const { restored, initialized } = await initializeApp(appStore, userStore)
        if (!restored) {
            next(loginRoute())
            return
        }
        next(initialized ? { ...to, replace: true } : undefined)
    } catch (error) {
        // 初始化失败不等于认证失效，保留凭据并停止本次自动恢复
        next(loginRoute())
    }
})

router.afterEach(() => {
    progress.done()
})
