import { createApp, h, reactive } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import antd, { ConfigProvider, theme } from 'ant-design-vue'
import i18n from '../../src/locales'
import { useUserStore, useAppStore } from '../../src/store'
import ProviderPanel from '../../src/views/resource/ProviderMonitorTab.vue'
import ModelPanel from '../../src/views/resource/ModelMonitorTab.vue'
import Chart from '../../src/components/Chart/Chart.vue'
import 'ant-design-vue/dist/reset.css'
import '../../src/styles/theme-variables.css'

const params = new URLSearchParams(location.search)
const dark = params.get('theme') === 'dark'
const model = params.get('kind') === 'model'
const target = reactive({ id: 'p-1', code: 'p-code', name: 'Provider', active: true })
window.detailMonitoringFixture = {
    select: (patch) => Object.assign(target, patch),
}
document.documentElement.dataset.theme = dark ? 'dark' : 'light'
document.body.style.cssText = `margin:0;background:${dark ? '#101520' : '#f1f5f9'};font-family:system-ui,sans-serif`
const app = createApp({
    render: () =>
        h(
            ConfigProvider,
            { theme: { algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm } },
            {
                default: () =>
                    h(
                        'main',
                        {
                            style: {
                                padding: '24px',
                                '--dashboard-panel': dark ? '#141926' : '#ffffff',
                                '--dashboard-panel-soft': dark ? '#101520' : '#f8fafc',
                                '--dashboard-text': dark ? '#e7ecf6' : '#111827',
                                '--dashboard-text-secondary': dark ? '#a3adc0' : '#64748b',
                            },
                        },
                        [
                            h(model ? ModelPanel : ProviderPanel, {
                                ...(model
                                    ? { modelId: target.id, modelCode: target.code }
                                    : { providerId: target.id, providerCode: target.code, providerName: target.name }),
                                active: target.active,
                            }),
                        ]
                    ),
            }
        ),
})
app.use(createPinia()).use(antd).use(i18n)
app.component('x-chart', Chart)
const user = useUserStore()
user.token = 'fixture-session'
user.userInfo = { id: 'ordinary-user', is_root: false }
user.userInfoVerified = true
useAppStore().config.theme = dark ? 'dark' : 'light'
i18n.global.locale.value = params.get('locale') === 'en' ? 'en-us' : 'zh-ch'
const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { render: () => null } }],
})
app.use(router)
await router.push('/')
await router.isReady()
app.mount('#app')
