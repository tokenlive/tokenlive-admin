import { createApp, h } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import antd, { ConfigProvider, theme } from 'ant-design-vue'
import i18n from '../../src/locales'
import { useUserStore, useAppStore } from '../../src/store'
import ProviderDetail from '../../src/views/resource/ProviderDetail.vue'
import 'ant-design-vue/dist/reset.css'
import '../../src/styles/theme-variables.css'

document.documentElement.dataset.theme = 'light'
document.body.style.cssText = 'margin:0;background:#f1f5f9;font-family:system-ui,sans-serif'
const app = createApp({
    render: () => h(ConfigProvider, { theme: { algorithm: theme.defaultAlgorithm } }, { default: () => h(RouterView) }),
})
app.use(createPinia()).use(antd).use(i18n)
const user = useUserStore()
user.token = 'fixture-session'
user.userInfo = { id: 'ordinary-user', is_root: false }
user.userInfoVerified = true
useAppStore().config.theme = 'light'
i18n.global.locale.value = 'zh-ch'
const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/providers/:id', name: 'providerDetail', component: ProviderDetail }],
})
app.use(router)
window.providerSummaryFixture = { open: (id) => router.push({ name: 'providerDetail', params: { id } }) }
await router.push({ name: 'providerDetail', params: { id: 'a' } })
await router.isReady()
app.mount('#app')
