<template>
    <a-config-provider
        :locale="antdLocale"
        :theme="theme">
        <router-view />
    </a-config-provider>
</template>

<script setup>
import { computed, watch } from 'vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import enUS from 'ant-design-vue/es/locale/en_US'
import { theme as antdTheme } from 'ant-design-vue'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/en'
import { useAppStore } from '@/store'
import { storeToRefs } from 'pinia'
import storage from '@/utils/storage'
import { config as appConfig } from '@/config'

const appStore = useAppStore()
const { config } = storeToRefs(appStore)

// 动态设置 html data-theme 属性，确保全局自定义的暗黑模式 Less 样式生效
watch(
    () => config.value.theme,
    (val) => {
        if (val === 'dark') {
            document.documentElement.setAttribute('data-theme', 'dark')
        } else {
            document.documentElement.removeAttribute('data-theme')
        }
    },
    { immediate: true }
)

const antdLocale = computed(() => {
    const lang = storage.local.getItem(appConfig('storage.lang')) || 'zh-ch'
    return lang === 'en-us' ? enUS : zhCN
})

dayjs.locale(antdLocale.value === enUS ? 'en' : 'zh-cn')

const theme = computed(() => {
    const isDark = config.value.theme === 'dark'
    return {
        algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
            // 品牌紫色主色 — 智能、科技、可信
            colorPrimary: '#7c5cfc',
            colorLink: '#7c5cfc',
            colorLinkHover: '#9578ff',
            borderRadius: 6,
            fontFamily:
                "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif",
            ...(isDark
                ? {
                      colorBgBase: '#0d0f14',
                      colorBgContainer: '#141722',
                      colorBgLayout: '#0d0f14',
                      colorBorder: 'rgba(255, 255, 255, 0.08)',
                      colorBorderSecondary: 'rgba(255, 255, 255, 0.05)',
                      colorTextBase: '#e8eaed',
                      colorTextSecondary: '#8e919c',
                  }
                : {}),
        },
        components: {
            List: {
                paddingContentHorizontalLG: 0,
            },
            Table: {
                paddingContentVerticalLG: 12,
                padding: 12,
            },
            Card: {
                paddingLG: 16,
            },
            Tabs: isDark
                ? {
                      tabsActiveColor: '#ffffff',
                      tabsHoverColor: '#9578ff',
                  }
                : {},
            // 暗色主题组件定制
            Layout: isDark
                ? {
                      colorBgHeader: '#0d0f14',
                      colorBgBody: '#0d0f14',
                      colorBgTrigger: '#141722',
                  }
                : {},
            Menu: isDark
                ? {
                      colorItemBg: '#0d0f14',
                      colorSubItemBg: '#0d0f14',
                      colorItemBgSelected: 'rgba(124, 92, 252, 0.15)',
                      colorItemBgActive: 'rgba(124, 92, 252, 0.1)',
                      colorItemTextSelected: '#e0dbff',
                      colorItemTextHover: '#9578ff',
                      colorActiveBarWidth: 3,
                  }
                : {},
            Dropdown: isDark
                ? {
                      colorBgElevated: '#141722',
                  }
                : {},
            Select: isDark
                ? {
                      colorBgElevated: '#141722',
                  }
                : {},
            Modal: isDark
                ? {
                      colorBgElevated: '#141722',
                  }
                : {},
            Drawer: isDark
                ? {
                      colorBgElevated: '#141722',
                  }
                : {},
            Popover: isDark
                ? {
                      colorBgElevated: '#141722',
                  }
                : {},
            Tooltip: isDark
                ? {
                      colorBgElevated: '#1a1d28',
                  }
                : {},
        },
    }
})
</script>

<style lang="less"></style>
