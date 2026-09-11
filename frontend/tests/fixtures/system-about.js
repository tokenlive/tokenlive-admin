import { createApp, h, markRaw } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import antd from 'ant-design-vue'
import { HomeOutlined, DesktopOutlined, ClusterOutlined, KeyOutlined } from '@ant-design/icons-vue'
import App from '../../src/App.vue'
import BasicLayout from '../../src/layouts/BasicLayout.vue'
import Scrollbar from '../../src/components/Scrollbar/Scrollbar.vue'
import Breadcrumb from '../../src/components/Breadcrumb/Breadcrumb.vue'
import i18n from '../../src/locales'
import { useAppStore, useRouterStore, useUserStore } from '../../src/store'
import 'ant-design-vue/dist/reset.css'
import '../../src/styles/theme-variables.css'
import '../../src/styles/index.less'

const app = createApp(App)
app.use(createPinia()).use(antd).use(i18n)
app.component('x-scrollbar', Scrollbar)
app.component('x-breadcrumb', Breadcrumb)

const appStore = useAppStore()
appStore.config.multiTab = false
useUserStore().userInfo = { name: 'Admin' }
const menuItems = Array.from({ length: 30 }, (_, i) => ({
    name: `menu-${i}`,
    path: '/demo',
    meta: {
        title: i === 0 ? '供应商' : `导航项目 ${i}`,
        icon: i === 29 ? undefined : markRaw([ClusterOutlined, HomeOutlined, DesktopOutlined, KeyOutlined][i % 4]),
        breadcrumb: [],
    },
}))
useRouterStore().menuList = [
    { name: 'resources', meta: { title: '资源管理' }, children: menuItems.slice(0, 10) },
    { name: 'governance', meta: { title: '治理策略' }, children: menuItems.slice(10, 20) },
    { name: 'system', meta: { title: '系统管理' }, children: menuItems.slice(20) },
]
const router = createRouter({
    history: createMemoryHistory(),
    routes: [
        {
            path: '/',
            component: BasicLayout,
            children: [
                {
                    path: 'demo',
                    name: 'demo',
                    meta: {
                        title: '供应商',
                        breadcrumb: [
                            { name: 'resources', meta: { title: '资源管理' } },
                            { name: 'menu-0', meta: { title: '供应商' } },
                        ],
                        openKeys: ['resources'],
                    },
                    component: { render: () => h('h1', '供应商') },
                },
            ],
        },
    ],
})
app.use(router)
await router.push('/demo')
await router.isReady()
app.mount('#app')

// Fixture-only controls; no production test hooks or real backend writes.
window.layoutTest = {
    setConfig: (config) => Object.assign(appStore.config, config),
    setLocale: (locale) => {
        i18n.global.locale.value = locale
    },
}
