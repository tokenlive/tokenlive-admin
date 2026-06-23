import { defineStore } from 'pinia'
import storage from '@/utils/storage'
import useRouterStore from './router'
import { config } from '@/config'

const defaultConfig = {
    layout: 'leftRight', // 页面布局【topBottom=上下布局，leftRight=左右布局】
    menuMode: 'side', // 菜单模式【side=侧边菜单，top=顶部菜单，mix=混合菜单】
    sideCollapsedWidth: 60,
    sideWidth: 220,
    headerHeight: 60,
    sideTheme: 'dark', // 侧边菜单主题【dark=暗色，light=亮色】
    headerTheme: 'light', // 侧边菜单主题【dark=暗色，light=亮色】
    theme: 'light', // 整体主题【dark=暗色，light=亮色】
    multiTab: true,
    multiTabHeight: 48,
    mainMargin: 16,
}

const useAppStore = defineStore('app', {
    name: 'useAppStore',
    state: () => {
        const storedConfig = storage.session.getItem(config('storage.config'), null)
        const activeConfig = { ...defaultConfig, ...(storedConfig || {}) }
        activeConfig.layout = 'leftRight'
        activeConfig.menuMode = 'side'
        return {
            complete: false,
            config: activeConfig,
        }
    },
    getters: {
        mainOffsetTop: (state) => {
            const multiTabHeight = state.config?.multiTab ? `${state.config.multiTabHeight}px` : '0px'
            return `calc(${state.config.headerHeight}px + ${multiTabHeight} + ${state.config.mainMargin}px)`
        },
        mainHeight: (state) => {
            const multiTabHeight = state.config?.multiTab ? `${state.config.multiTabHeight}px` : '0px'
            return `calc(100vh - ${state.config.headerHeight}px - ${multiTabHeight} - ${state.config.mainMargin * 2}px)`
        },
    },
    actions: {
        /**
         * 初始化
         * @returns {Promise}
         */
        init() {
            const routerStore = useRouterStore()
            return new Promise((resolve) => {
                Promise.all([routerStore.getRouterList()])
                    .then(() => {
                        this.complete = true
                        resolve()
                    })
                    .catch(() => {})
            })
        },
        /**
         * 更新 config
         */
        updateConfig() {
            storage.session.setItem(config('storage.config'), this.config)
        },
        /**
         * 同步切换整体、顶部和侧边主题
         */
        toggleTheme() {
            const nextTheme = this.config.theme === 'dark' ? 'light' : 'dark'
            this.config.theme = nextTheme
            this.config.headerTheme = nextTheme
            this.config.sideTheme = nextTheme
            this.updateConfig()
        },
    },
})

export default useAppStore
