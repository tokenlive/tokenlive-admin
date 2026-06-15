<template>
    <div :class="['user-layout-container', { 'user-layout--dark': appStore.config.theme === 'dark' }]">
        <div class="user-layout-aside">
            <div class="aside-header">
                <h1>{{ title }}</h1>
            </div>
            <div class="aside-body">
                <img
                    alt=""
                    :src="assets('logos.png')" />
                <h3>{{ $t('pages.layouts.userLayout.title') }}</h3>
                <!--                <p>Vue3 + Ant Design Vue + vite</p>-->
            </div>
            <div class="aside-footer">
                <p>© {{ title }} {{ version }}</p>
            </div>
        </div>
        <div class="user-layout-main">
            <div class="user-layout-content">
                <div class="user-layout-top">
                    <div class="user-layout-header">{{ $t('login') }}</div>
                    <!--                    <div class="user-layout-desc">欢迎使用{{ title }}</div>-->
                </div>
                <div class="user-layout-form">
                    <router-view></router-view>
                </div>
            </div>
        </div>

        <div
            class="basic-header__right"
            style="padding: 30px">
            <a-space :size="16">
                <a-tooltip :title="themeToggleTitle">
                    <action-button @click="handleThemeToggle">
                        <bulb-filled v-if="appStore.config.theme === 'dark'" />
                        <bulb-outlined v-else />
                    </action-button>
                </a-tooltip>
                <a-dropdown :trigger="['hover']">
                    <action-button :style="{ height: '44px' }">
                        <translation-outlined />
                    </action-button>
                    <template #overlay>
                        <a-menu v-model:selectedKeys="current">
                            <a-menu-item
                                v-for="(item, key) in langData"
                                :key="key"
                                @click="handleLang(key)">
                                {{ item.icon }} {{ item.label }}
                            </a-menu-item>
                        </a-menu>
                    </template>
                </a-dropdown>
            </a-space>
        </div>
    </div>
</template>

<script setup>
import { assets } from '@/utils/util'
import { config as conf, config } from '@/config'
import { computed, ref } from 'vue'
import { BulbFilled, BulbOutlined, TranslationOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'
import ActionButton from './components/ActionButton.vue'

import storage from '@/utils/storage'
import { useI18n } from 'vue-i18n'
const { locale, t } = useI18n()
const appStore = useAppStore()
defineOptions({
    name: 'UserLayout',
})

const { version } = __APP_INFO__
const title = config('app.title')
const defaultLang = storage.local.getItem(conf('storage.lang')) || 'zh-ch'
const current = ref(defaultLang)
const themeToggleTitle = computed(() =>
    t(appStore.config.theme === 'dark' ? 'app.setting.theme.switch.light' : 'app.setting.theme.switch.dark')
)
const langData = ref({
    'zh-ch': {
        lang: 'zh-ch',
        label: '简体中文',
        icon: '🇨🇳',
        title: '语言',
    },
    'en-us': {
        lang: 'en-us',
        label: 'English',
        icon: '🇺🇸',
        title: 'Language',
    },
})

/**
 * 切换语言
 */

function handleLang(lang) {
    storage.local.setItem(conf('storage.lang'), lang)
    locale.value = lang
    current.value = lang
    location.reload()
}

function handleThemeToggle() {
    appStore.toggleTheme()
}
</script>

<style lang="less" scoped>
.user-layout {
    &-container {
        min-height: 100vh;
        background-color: #fff;
        background-repeat: no-repeat;
        background-position: center 110px;
        background-size: 100%;
        color: rgba(0, 0, 0, 0.88);
        display: flex;
        transition:
            background-color 0.3s ease,
            color 0.3s ease;
    }

    &-aside {
        width: 538px;
        flex: 0 0 538px;
        display: flex;
        flex-direction: column;
        background: #235bda url('@/assets/login_aside_bg.jpg') no-repeat left top / 100% auto;
        position: relative;

        .aside {
            &-header {
                display: flex;
                flex-direction: column;
                padding: 48px;

                h1 {
                    font-size: 20px;
                    font-weight: 500;
                    color: #fff;
                }
            }

            &-body {
                flex: 1;
                text-align: center;
                padding: 48px 0 0;

                img {
                    width: 80%;
                }

                h3 {
                    color: #fff;
                }

                p {
                    color: rgba(255, 255, 255, 0.85);
                }
            }

            &-footer {
                color: rgba(255, 255, 255, 0.65);
                font-size: 12px;
                padding: 48px;
            }
        }
    }

    &-main {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 64px 0 144px;
    }

    &-content {
        width: 368px;
        height: 440px;
    }

    &-header {
        display: flex;
        align-items: center;
        font-size: 30px;
        font-weight: 500;
    }

    &-desc {
        margin: 8px 0 24px;
    }
}

.basic-header__right {
    color: var(--color-text-primary);
}

// 暗色主题仅调整颜色，保持与亮色主题相同的布局结构
.user-layout-container.user-layout--dark {
    background-color: var(--color-bg-base);
    color: var(--color-text-primary);

    .user-layout-header {
        color: var(--color-text-primary);
    }

    .basic-header__right {
        :deep(.anticon) {
            color: var(--color-text-secondary) !important;
            &:hover {
                color: var(--color-text-primary) !important;
            }
        }
    }
}
</style>
