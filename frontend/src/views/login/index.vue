<template>
    <div class="login-panel">
        <a-tabs class="login-panel__tabs">
            <!-- 账号登录 -->
            <a-tab-pane
                key="account"
                :tab="$t('pages.login.accountLogin.tab')">
                <a-form
                    :model="formData"
                    :rules="formRules"
                    ref="formRef">
                    <a-form-item name="username">
                        <a-input
                            :placeholder="$t('pages.login.username.placeholder')"
                            v-model:value="formData.username"
                            size="large">
                            <template #prefix>
                                <user-outlined></user-outlined>
                            </template>
                        </a-input>
                    </a-form-item>
                    <a-form-item name="password">
                        <a-input
                            v-model:value="formData.password"
                            size="large"
                            type="password"
                            :placeholder="$t('pages.login.password.placeholder')"
                            @pressEnter="handleLogin">
                            <template #prefix>
                                <lock-outlined></lock-outlined>
                            </template>
                        </a-input>
                    </a-form-item>
                    <a-form-item name="captcha_code">
                        <a-space class="login-panel__captcha">
                            <a-input
                                v-model:value="formData.captcha_code"
                                size="large"
                                type="text"
                                :placeholder="$t('pages.login.captcha.placeholder')"
                                @pressEnter="handleLogin">
                                <template #prefix>
                                    <safety-outlined />
                                </template>
                            </a-input>
                            <a-image
                                class="login-panel__captcha-image"
                                @click="getCaptcha"
                                :preview="false"
                                :width="140"
                                :height="42"
                                :src="captcha_img" />
                        </a-space>
                    </a-form-item>
                    <a-form-item name="rememberMe">
                        <a-checkbox v-model:checked="formData.rememberMe">
                            {{ $t('pages.login.rememberMe') }}
                        </a-checkbox>
                    </a-form-item>
                    <a-form-item>
                        <a-button
                            type="primary"
                            size="large"
                            block
                            :loading="loading"
                            @click="handleLogin">
                            {{ $t('pages.login.submit') }}
                        </a-button>
                    </a-form-item>
                </a-form>
                <div
                    v-if="oauthProviders.length"
                    class="login-panel__oauth">
                    <a-divider>
                        {{ $t('pages.login.oauth.divider') }}
                    </a-divider>
                    <a-space
                        direction="vertical"
                        class="login-panel__oauth-actions">
                        <a-button
                            v-for="provider in oauthProviders"
                            :key="provider.provider"
                            size="large"
                            block
                            class="login-panel__oauth-button"
                            @click="handleOAuthLogin(provider)">
                            <template #icon>
                                <google-outlined v-if="provider.provider === 'google'" />
                                <github-outlined v-else-if="provider.provider === 'github'" />
                            </template>
                            {{ oauthProviderText(provider.provider) }}
                        </a-button>
                    </a-space>
                </div>
            </a-tab-pane>
        </a-tabs>
    </div>
</template>

<script setup>
import { Modal } from 'ant-design-vue'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { GithubOutlined, GoogleOutlined, LockOutlined, SafetyOutlined, UserOutlined } from '@ant-design/icons-vue'
import { config } from '@/config'
import { useForm } from '@/hooks'
import { useAppStore, useRouterStore, useUserStore } from '@/store'
import apis from '@/apis'
import { md5 } from 'js-md5'
import { useI18n } from 'vue-i18n'
defineOptions({
    name: 'login',
})
const { t } = useI18n() // 解构出t方法
const { formData, formRef, formRules } = useForm()
const appStore = useAppStore()
const userStore = useUserStore()
const routerStore = useRouterStore()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const oauthLoading = ref(false)
const oauthProviders = ref([])
const captcha_img = ref('')
const captcha_id = ref('')
const httpApi = import.meta.env.BASE_URL + `api/v1/captcha/image`
const redirect = computed(() => decodeURIComponent(route.query?.redirect ?? ''))
const oauthTicket = computed(() => route.query?.oauth_ticket ?? '')

formRules.value = {
    username: { required: true, message: t('pages.login.username.placeholder') },
    password: { required: true, message: t('pages.login.password.placeholder') },
    captcha_code: { required: true, message: t('pages.login.captcha.placeholder') },
}

// 初始化表单数据，记住我默认不勾选
formData.value.rememberMe = false

onMounted(() => {
    // 清理登录信息
    userStore.logout()
    loadOAuthProviders()
    if (oauthTicket.value) {
        handleOAuthExchange()
    }
    getCaptcha()
})

async function loadOAuthProviders() {
    const { success, data } = await apis.user.oauthProviders().catch(() => ({}))
    if (config('http.code.success') === success && Array.isArray(data)) {
        oauthProviders.value = data
    }
}

function oauthProviderText(provider) {
    if (provider === 'google') return 'Continue with Google'
    if (provider === 'github') return 'Continue with GitHub'
    return `Continue with ${provider}`
}

async function handleOAuthExchange() {
    oauthLoading.value = true
    const { success } = await userStore.oauthExchange(oauthTicket.value).catch(() => {
        oauthLoading.value = false
    })
    oauthLoading.value = false
    if (config('http.code.success') === success) {
        if (appStore.complete) {
            goIndex()
        } else {
            await appStore.init()
            goIndex()
        }
    }
}

function handleOAuthLogin(provider) {
    const target = new URL(provider.login_url, window.location.origin)
    if (redirect.value) {
        target.searchParams.set('redirect', redirect.value)
    }
    window.location.href = target.toString()
}

/**
 * 获取验证码
 */
async function getCaptcha() {
    const { data } = await apis.common.getCaptcha().catch(() => {})
    captcha_id.value = data?.captcha_id
    captcha_img.value = httpApi + `?id=${data?.captcha_id}`
}
/**
 * 登录
 * @return {Promise<void>}
 */
async function handleLogin() {
    formRef.value.validate().then(async (values) => {
        values.captcha_id = captcha_id.value
        // 添加 remember_me 字段
        values.remember_me = formData.value.rememberMe
        if (values.password === 'admin') values.password = md5(values.password)

        loading.value = true
        const { success } = await userStore
            .login({
                ...values,
            })
            .catch(() => {
                loading.value = false
                getCaptcha()
            })
        loading.value = false
        if (config('http.code.success') === success) {
            // 加载完成
            if (appStore.complete) {
                goIndex()
            } else {
                await appStore.init()
                goIndex()
            }
        }
    })
}

/**
 * 获取首页路由
 * @return {*}
 */
function getFirstValidRoute() {
    const indexRoute = routerStore.indexRoute
    if (!indexRoute) {
        Modal.warning({
            title: '系统提示',
            content: '没有任何权限，请联系系统管理员',
            onOk: () => {
                window.location.reload()
            },
        })
    }
    return indexRoute
}

/**
 * 去首页
 */
function goIndex() {
    if (redirect.value) {
        location.href = redirect.value
    } else {
        const indexRoute = getFirstValidRoute()
        if (!indexRoute) return
        router.push(indexRoute)
    }
}
</script>

<style lang="less" scoped>
.login-panel {
    :deep(.ant-tabs-nav) {
        display: none;
    }

    :deep(.ant-tabs-content-holder) {
        overflow: visible;
    }

    :deep(.ant-form-item) {
        margin-bottom: 18px;
    }

    :deep(.ant-form-item-explain) {
        display: none;
    }

    :deep(.ant-input-affix-wrapper) {
        height: 46px;
        border-color: rgba(255, 255, 255, 0.12);
        border-radius: 14px;
        background: rgba(255, 255, 255, 0.08);
        box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.12);
        transition:
            border-color 0.2s ease,
            box-shadow 0.2s ease,
            background 0.2s ease;
    }

    :deep(.ant-input-affix-wrapper:hover),
    :deep(.ant-input-affix-wrapper-focused) {
        border-color: rgba(104, 213, 196, 0.52);
        background: rgba(255, 255, 255, 0.12);
        box-shadow: 0 0 0 3px rgba(104, 213, 196, 0.1);
    }

    :deep(.ant-input) {
        color: rgba(245, 249, 255, 0.88);
        background: transparent;
    }

    :deep(.ant-input::placeholder) {
        color: rgba(245, 249, 255, 0.5);
    }

    :deep(.ant-input-prefix) {
        margin-inline-end: 10px;
        color: #68d5c4;
    }

    :deep(.ant-checkbox-wrapper) {
        color: rgba(245, 249, 255, 0.72);
        font-size: 13px;
    }

    :deep(.ant-btn-primary) {
        height: 46px;
        border: none;
        border-radius: 14px;
        background: linear-gradient(135deg, #147fc7 0%, #13b69d 100%);
        box-shadow: 0 14px 28px rgba(20, 127, 199, 0.22);
        font-weight: 600;
    }

    :deep(.ant-btn-primary:hover),
    :deep(.ant-btn-primary:focus-visible) {
        background: linear-gradient(135deg, #0f70b5 0%, #0fa98f 100%);
        box-shadow: 0 16px 34px rgba(20, 127, 199, 0.3);
    }

    :deep(.ant-divider) {
        margin: 8px 0 16px;
        color: rgba(245, 249, 255, 0.48);
        font-size: 12px;
    }

    :deep(.ant-divider-horizontal.ant-divider-with-text::before),
    :deep(.ant-divider-horizontal.ant-divider-with-text::after) {
        border-color: rgba(255, 255, 255, 0.14);
    }

    &__captcha {
        width: 100%;

        :deep(.ant-space-item:first-child) {
            flex: 1;
            min-width: 0;
        }
    }

    &__captcha-image {
        cursor: pointer;

        :deep(.ant-image-img) {
            border-radius: 14px;
            box-shadow: 0 10px 24px rgba(30, 80, 110, 0.12);
        }
    }

    &__oauth-actions {
        width: 100%;
    }

    &__oauth-button {
        height: 46px;
        border-color: rgba(255, 255, 255, 0.14);
        border-radius: 14px;
        color: rgba(245, 249, 255, 0.82);
        background: rgba(255, 255, 255, 0.08);
        font-weight: 600;

        &:hover,
        &:focus-visible {
            border-color: rgba(104, 213, 196, 0.52);
            color: #ffffff;
            background: rgba(255, 255, 255, 0.12);
        }
    }
}

:global(.user-layout--dark) {
    .login-panel {
        :deep(.ant-input-affix-wrapper) {
            border-color: rgba(255, 255, 255, 0.12);
            background: rgba(255, 255, 255, 0.08);
        }

        :deep(.ant-input-affix-wrapper:hover),
        :deep(.ant-input-affix-wrapper-focused) {
            border-color: rgba(104, 213, 196, 0.52);
            background: rgba(255, 255, 255, 0.12);
            box-shadow: 0 0 0 3px rgba(104, 213, 196, 0.1);
        }

        :deep(.ant-input),
        :deep(.ant-input::placeholder),
        :deep(.ant-checkbox-wrapper) {
            color: rgba(245, 249, 255, 0.72);
        }

        :deep(.ant-input-prefix) {
            color: #68d5c4;
        }
    }
}
</style>
