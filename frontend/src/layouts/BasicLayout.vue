<template>
    <a-layout class="layout">
        <template #default>
            <!-- 上下布局 -->
            <template v-if="config.layout === 'topBottom'">
                <!-- 侧边菜单 -->
                <template v-if="config.menuMode === 'side'">
                    <basic-header
                        :has-update="hasUpdate"
                        :theme="config.headerTheme"
                        @config="$refs.configDialogRef.handleOpen()">
                        <template #left>
                            <brand :theme="config.headerTheme"></brand>
                            <x-breadcrumb :theme="config.headerTheme"></x-breadcrumb>
                        </template>
                    </basic-header>
                    <a-layout>
                        <basic-side
                            :theme="config.sideTheme"
                            :style="{
                                height: `calc(100vh - ${config.headerHeight}px)`,
                                top: `${config.headerHeight}px`,
                            }">
                            <basic-menu
                                :theme="config.sideTheme"
                                :data-list="sideMenuList"></basic-menu>
                            <template #footer="{ collapsed }">
                                <sidebar-version
                                    :version="displayVersion"
                                    :has-update="hasUpdate"
                                    :collapsed="collapsed"
                                    :theme="config.sideTheme"
                                    @about="aboutOpen = true" />
                            </template>
                        </basic-side>
                        <a-layout>
                            <multi-tab v-if="config.multiTab"></multi-tab>
                            <basic-content></basic-content>
                        </a-layout>
                    </a-layout>
                </template>

                <!-- 混合菜单 -->
                <template v-if="config.menuMode === 'mix'">
                    <basic-header
                        :has-update="hasUpdate"
                        :theme="config.headerTheme"
                        @config="$refs.configDialogRef.handleOpen()">
                        <template #left>
                            <brand :theme="config.headerTheme"></brand>
                        </template>
                        <basic-menu
                            mode="horizontal"
                            :theme="config.headerTheme"
                            :data-list="topMenuList"></basic-menu>
                    </basic-header>
                    <a-layout>
                        <template v-if="sideMenuList.length">
                            <basic-side
                                :theme="config.sideTheme"
                                :style="{
                                    height: `calc(100vh - ${config.headerHeight}px)`,
                                    top: `${config.headerHeight}px`,
                                }">
                                <basic-menu
                                    :theme="config.sideTheme"
                                    :data-list="sideMenuList"></basic-menu>
                                <template #footer="{ collapsed }">
                                    <sidebar-version
                                        :version="displayVersion"
                                        :has-update="hasUpdate"
                                        :collapsed="collapsed"
                                        :theme="config.sideTheme"
                                        @about="aboutOpen = true" />
                                </template>
                            </basic-side>
                        </template>
                        <a-layout>
                            <multi-tab v-if="config.multiTab"></multi-tab>
                            <basic-content></basic-content>
                        </a-layout>
                    </a-layout>
                </template>
            </template>

            <!-- 左右布局 -->
            <template v-if="config.layout === 'leftRight'">
                <!-- 侧边菜单 -->
                <template v-if="config.menuMode === 'side'">
                    <basic-side
                        :theme="config.sideTheme"
                        :style="{
                            height: `100vh`,
                            top: 0,
                        }">
                        <template #header>
                            <brand :theme="config.sideTheme"></brand>
                        </template>
                        <basic-menu
                            :theme="config.sideTheme"
                            :data-list="sideMenuList"></basic-menu>
                        <template #footer="{ collapsed }">
                            <sidebar-version
                                :version="displayVersion"
                                :has-update="hasUpdate"
                                :collapsed="collapsed"
                                :theme="config.sideTheme"
                                @about="aboutOpen = true" />
                        </template>
                    </basic-side>
                    <a-layout>
                        <basic-header
                            :has-update="hasUpdate"
                            :theme="config.headerTheme"
                            @config="$refs.configDialogRef.handleOpen()">
                            <template #left>
                                <x-breadcrumb :theme="config.headerTheme"></x-breadcrumb>
                            </template>
                        </basic-header>
                        <multi-tab v-if="config.multiTab"></multi-tab>
                        <basic-content></basic-content>
                    </a-layout>
                </template>
                <!-- 混合菜单 -->
                <template v-if="config.menuMode === 'mix'">
                    <basic-side
                        :theme="config.sideTheme"
                        :style="{
                            height: `100vh`,
                            top: 0,
                        }">
                        <template #header>
                            <brand :theme="config.sideTheme"></brand>
                        </template>
                        <basic-menu
                            :theme="config.sideTheme"
                            :data-list="sideMenuList"></basic-menu>
                        <template #footer="{ collapsed }">
                            <sidebar-version
                                :version="displayVersion"
                                :has-update="hasUpdate"
                                :collapsed="collapsed"
                                :theme="config.sideTheme"
                                @about="aboutOpen = true" />
                        </template>
                    </basic-side>
                    <a-layout>
                        <basic-header
                            :has-update="hasUpdate"
                            :theme="config.headerTheme"
                            @config="$refs.configDialogRef.handleOpen()">
                            <basic-menu
                                mode="horizontal"
                                :theme="config.headerTheme"
                                :data-list="topMenuList"></basic-menu>
                        </basic-header>
                        <multi-tab v-if="config.multiTab"></multi-tab>
                        <basic-content></basic-content>
                    </a-layout>
                </template>
            </template>

            <!-- 顶部菜单，不区分布局方式 -->
            <template v-if="config.menuMode === 'top'">
                <basic-header
                    :has-update="hasUpdate"
                    :theme="config.headerTheme"
                    @config="$refs.configDialogRef.handleOpen()">
                    <template #left>
                        <brand :theme="config.headerTheme"></brand>
                    </template>
                    <basic-menu
                        mode="horizontal"
                        :theme="config.headerTheme"
                        :data-list="topMenuList"></basic-menu>
                </basic-header>
                <multi-tab v-if="config.multiTab"></multi-tab>
                <basic-content></basic-content>
            </template>
        </template>
    </a-layout>

    <config-dialog
        ref="configDialogRef"
        @about="aboutOpen = true"></config-dialog>
    <system-about-dialog
        v-model:open="aboutOpen"
        :summary="summary"
        :updates="updates"
        :fallback-version="fallbackVersion"
        :checking="checking"
        :retry-after-seconds="retryAfterSeconds"
        :error="error"
        :labels="versionLabels"
        @check="check" />
</template>

<script setup>
import { storeToRefs } from 'pinia'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSystemVersion } from '@/composables/useSystemVersion'
import { formatIdentity, hasAvailableUpdate } from '@/utils/system-version'
import { useAppStore } from '@/store'
import useMultiTab from './hooks/useMultiTab'
import useMenu from './hooks/useMenu'
import BasicContent from './components/BasicContent.vue'
import BasicHeader from './components/BasicHeader.vue'
import BasicMenu from './components/BasicMenu.vue'
import BasicSide from './components/BasicSide.vue'
import Brand from './components/Brand.vue'
import MultiTab from './components/MultiTab.vue'
import ConfigDialog from './components/ConfigDialog.vue'
import SidebarVersion from './components/SidebarVersion.vue'
import SystemAboutDialog from './components/SystemAboutDialog.vue'

defineOptions({
    name: 'BasicLayout',
})

useMultiTab()
const appStore = useAppStore()
const { sideMenuList, topMenuList } = useMenu()

const { config } = storeToRefs(appStore)

const configDialogRef = ref()
const aboutOpen = ref(false)
const { t } = useI18n()
// All menu layouts and About share this one cache-reading owner.
const { summary, updates, fallbackVersion, checking, error, retryAfterSeconds, load, check } = useSystemVersion()
const versionLabels = computed(() => ({
    professional: t('app.about.edition.professional'),
    standalone: t('app.about.edition.standalone'),
    unknown: t('app.about.unknown'),
    version: t('app.about.version'),
    channel: t('app.about.channel'),
    build: t('app.about.build'),
    separator: t('app.about.labelSeparator'),
    channelValues: {
        release: t('app.about.channel.release'),
        homebrew: t('app.about.channel.homebrew'),
        unknown: t('app.about.unknown'),
    },
    buildValues: {
        release: t('app.about.build.release'),
        dev: t('app.about.build.dev'),
        unknown: t('app.about.unknown'),
    },
}))
const displayVersion = computed(() =>
    summary.value
        ? formatIdentity(summary.value.identity, versionLabels.value)
        : t('app.about.frontendBuildShort', { version: fallbackVersion.value })
)
const hasUpdate = computed(() => hasAvailableUpdate(summary.value, updates.value))
watch(aboutOpen, (open) => {
    if (open) void load()
})
</script>

<style lang="less" scoped>
.layout {
    min-height: 100vh;
}
</style>
