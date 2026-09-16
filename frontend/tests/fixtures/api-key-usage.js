import { createApp, h, ref } from 'vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import antd, { ConfigProvider, theme } from 'ant-design-vue'
import i18n from '../../src/locales'
import { useUserStore } from '../../src/store'
import Panel from '../../src/views/home/components/APIKeyUsageRanking.vue'
import Home from '../../src/views/home/index.vue'
import Chart from '../../src/components/Chart/Chart.vue'
import 'ant-design-vue/dist/reset.css'
import '../../src/styles/theme-variables.css'

const params = new URLSearchParams(location.search)
const dark = params.get('theme') === 'dark'
const full = params.get('full') === '1'
const authorized = ref(params.get('root') !== '0')
document.documentElement.dataset.theme = dark ? 'dark' : 'light'
document.body.style.cssText = `margin:0;background:${dark ? '#101520' : '#f1f5f9'};font-family:system-ui,sans-serif`
const wrapperStyle = {
    padding: '24px',
    '--dashboard-panel': dark ? '#141926' : '#ffffff',
    '--dashboard-panel-soft': dark ? '#101520' : '#f8fafc',
    '--dashboard-text': dark ? '#e7ecf6' : '#111827',
    '--dashboard-text-secondary': dark ? '#a3adc0' : '#64748b',
    '--dashboard-accent': '#2f8cff',
}
// Test fixture only: never connect the real homepage websocket to a backend.
window.WebSocket = class {
    static OPEN = 1
    readyState = 3
    constructor() {
        queueMicrotask(() => this.onclose?.())
    }
    close() {}
    send() {}
}
const app = createApp({
    render: () =>
        h(
            ConfigProvider,
            { theme: { algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm } },
            {
                default: () =>
                    h('main', { style: wrapperStyle }, [
                        full
                            ? h(Home)
                            : authorized.value
                              ? h(Panel, { authorized: authorized.value })
                              : h('div', { id: 'usage-denied' }, 'No ranking permission'),
                    ]),
            }
        ),
})
const pinia = createPinia()
app.use(pinia).use(antd).use(i18n)
app.component('x-chart', Chart)
const user = useUserStore()
user.token = 'fixture-session'
user.refreshToken = ''
user.userInfo = { id: 'fixture-root', is_root: authorized.value }
user.userInfoVerified = params.get('verify') !== '1'
i18n.global.locale.value = params.get('locale') === 'en-us' ? 'en-us' : 'zh-ch'
const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { render: () => null } }],
})
app.use(router)
await router.push('/')
await router.isReady()
app.mount('#app')
window.usageTest = {
    setAuthorized(value) {
        authorized.value = value
        user.userInfo = { id: value ? 'fixture-root' : 'ordinary', is_root: value }
        user.userInfoVerified = true
    },
}
