<template>
    <a-drawer
        v-model:open="open"
        :title="$t('app.setting.pagestyle')"
        :closable="false"
        :width="360">
        <a-form
            label-align="left"
            :colon="false"
            :label-col="{ span: 24 }">
            <!--            <div class="mb-8-2 fw-600">菜单设置</div>-->

            <a-form-item
                :label="$t('app.setting.theme')"
                class="mb-8-2"
                :label-col="{ flex: 'auto' }"
                :wrapper-col="{ style: { flex: '0 0 auto' } }">
                <a-select
                    v-model:value="config.theme"
                    :options="themeList"
                    @change="onChange"></a-select>
            </a-form-item>
            <a-form-item
                :label="$t('app.setting.topBottom')"
                class="mb-8-2"
                :label-col="{ flex: 'auto' }"
                :wrapper-col="{ style: { flex: '0 0 auto' } }">
                <a-select
                    v-model:value="config.headerTheme"
                    :options="themeList"
                    @change="onChange"></a-select>
            </a-form-item>
            <a-form-item
                :label="$t('app.setting.leftRight')"
                class="mb-8-2"
                :label-col="{ flex: 'auto' }"
                :wrapper-col="{ style: { flex: '0 0 auto' } }">
                <a-select
                    v-model:value="config.sideTheme"
                    :options="themeList"
                    @change="onChange"></a-select>
            </a-form-item>
            <!--            <a-divider></a-divider>-->
            <!--            <div class="mb-8-2 fw-600">内容区域</div>-->
            <!--            <a-form-item-->
            <!--                label="标签页"-->
            <!--                :label-col="{ flex: 'auto' }"-->
            <!--                :wrapper-col="{ style: { flex: '0 0 auto' } }">-->
            <!--                <a-switch-->
            <!--                    v-model:checked="config.multiTab"-->
            <!--                    size="small"-->
            <!--                    @change="onChange"></a-switch>-->
            <!--            </a-form-item>-->
        </a-form>
        <template #footer>
            <a-button
                type="text"
                block
                aria-haspopup="dialog"
                @click="handleAbout">
                <template #icon><info-circle-outlined /></template>
                {{ $t('app.about.title') }}
            </a-button>
        </template>
    </a-drawer>
</template>

<script setup>
import { storeToRefs } from 'pinia'
import { ref } from 'vue'
import { useAppStore } from '@/store'
import { useI18n } from 'vue-i18n'
import { InfoCircleOutlined } from '@ant-design/icons-vue'
const emit = defineEmits(['about'])
const { t } = useI18n()
const appStore = useAppStore()

const { config } = storeToRefs(appStore)

const open = ref(false)
let triggerElement = null
const themeList = ref([
    { value: 'light', label: t('app.setting.pagestyle.light') },
    { value: 'dark', label: t('app.setting.pagestyle.dark') },
])

function handleOpen() {
    triggerElement = document.activeElement
    open.value = true
}

function handleAbout() {
    open.value = false
    // Let the modal remember the visible Settings trigger, not this closing drawer.
    triggerElement?.focus({ preventScroll: true })
    emit('about')
}

function onChange() {
    appStore.updateConfig()
}

defineExpose({
    handleOpen,
})
</script>

<style lang="less" scoped></style>
