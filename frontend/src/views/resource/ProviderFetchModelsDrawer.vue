<template>
    <a-drawer
        :open="visible"
        :title="$t('pages.provider.fetchModels.title')"
        :width="660"
        :after-open="onAfterOpen"
        @close="handleClose">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            label-align="left"
            :colon="false"
            layout="horizontal">
            <a-form-item
                name="space_code"
                :label="$t('pages.model.form.space_code')"
                :label-col="{ flex: '110px' }"
                :wrapper-col="{ flex: 'auto' }"
                :rules="formRules.space_code">
                <a-select
                    v-model:value="formData.space_code"
                    :placeholder="$t('pages.model.form.space_code.placeholder')"
                    show-search
                    :filter-option="filterSpaceOption"
                    style="width: 100%">
                    <a-select-option
                        v-for="item in spaceOptions"
                        :key="item.code"
                        :value="item.code">
                        {{ item.name }} ({{ item.code }})
                    </a-select-option>
                </a-select>
            </a-form-item>

            <a-form-item
                name="base_url"
                :label="$t('pages.provider.fetchModels.base_url')"
                :label-col="{ flex: '110px' }"
                :wrapper-col="{ flex: 'auto' }"
                :rules="formRules.base_url">
                <a-input
                    v-model:value="formData.base_url"
                    :placeholder="$t('pages.provider.fetchModels.base_url.placeholder')" />
            </a-form-item>

            <a-form-item
                name="api_key"
                :label="$t('pages.provider.fetchModels.api_key')"
                :label-col="{ flex: '110px' }"
                :wrapper-col="{ flex: 'auto' }"
                style="align-items: center">
                <a-row
                    :gutter="8"
                    align="middle">
                    <a-col :span="18">
                        <a-auto-complete
                            v-model:value="formData.api_key"
                            :options="apiKeyOptions"
                            allow-clear
                            :placeholder="$t('pages.provider.fetchModels.api_key.placeholder')"
                            style="width: 100%" />
                    </a-col>
                    <a-col :span="6">
                        <a-button
                            type="primary"
                            :loading="fetching"
                            block
                            @click="handleFetchModels">
                            <template #icon>
                                <cloud-download-outlined />
                            </template>
                            {{ $t('pages.provider.fetchModels.fetch') }}
                        </a-button>
                    </a-col>
                </a-row>
            </a-form-item>
        </a-form>

        <a-divider />

        <div class="model-table-container">
            <div
                class="model-list-header"
                style="margin-bottom: 12px">
                <span class="model-count">
                    {{
                        $t('pages.provider.fetchModels.selected', {
                            count: selectedModels.length,
                            total: filteredModelList.length,
                        })
                    }}
                </span>
            </div>
            <a-input-search
                v-model:value="searchQuery"
                placeholder="输入发现的模型名称/Code进行模糊过滤..."
                style="margin-bottom: 12px; width: 100%"
                allow-clear />
            <a-table
                :columns="columns"
                :data-source="filteredModelList"
                :row-selection="rowSelection"
                :row-key="(record) => record.id"
                :pagination="false"
                :scroll="{ y: 'calc(100vh - 365px)' }"
                size="small"
                class="model-table">
            </a-table>
        </div>

        <template #footer>
            <a-space>
                <a-button @click="handleClose">{{ $t('button.cancel') }}</a-button>
                <a-button
                    type="primary"
                    :disabled="selectedModels.length === 0"
                    @click="handleConfirm">
                    {{ $t('pages.provider.fetchModels.confirm', { count: selectedModels.length }) }}
                </a-button>
            </a-space>
        </template>
    </a-drawer>
</template>

<script setup>
import { ref, computed, h } from 'vue'
import { CloudDownloadOutlined } from '@ant-design/icons-vue'
import { cloneDeep } from 'lodash-es'
import apis from '@/apis'
import { config } from '@/config'
import { useI18n } from 'vue-i18n'

const emit = defineEmits(['confirm'])

const { t } = useI18n()
const visible = ref(false)
const fetching = ref(false)
const fetched = ref(false)
const formRef = ref(null)
const providerId = ref('')
const providerApiKeys = ref([])

const apiKeyOptions = computed(() => {
    return providerApiKeys.value.map((key) => ({
        value: key,
        label: maskKey(key),
    }))
})

const spaceOptions = ref([])

const formData = ref({
    space_code: undefined,
    base_url: '',
    api_key: undefined,
})

const formRules = ref({
    space_code: [{ required: true, message: t('pages.model.form.space_code.placeholder') }],
    base_url: [{ required: true, message: t('pages.provider.fetchModels.base_url.required') }],
})

const modelList = ref([])
const selectedModels = ref([])
const searchQuery = ref('')

const filteredModelList = computed(() => {
    if (!searchQuery.value) return modelList.value
    const query = searchQuery.value.toLowerCase()
    return modelList.value.filter(
        (m) => m.id.toLowerCase().includes(query) || (m.owned_by && m.owned_by.toLowerCase().includes(query))
    )
})

const columns = computed(() => [
    {
        title: t('pages.provider.fetchModels.table.model'),
        dataIndex: 'id',
        key: 'id',
        width: 380,
        customRender: ({ text }) => {
            return h('span', { class: 'model-id' }, text)
        },
    },
    {
        title: t('pages.provider.fetchModels.table.owned_by'),
        dataIndex: 'owned_by',
        key: 'owned_by',
        width: 180,
    },
])

const rowSelection = computed(() => {
    return {
        selectedRowKeys: selectedModels.value,
        onChange: (selectedRowKeys) => {
            selectedModels.value = selectedRowKeys
        },
    }
})

function maskKey(key) {
    if (!key || key.length <= 8) return key
    return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}

async function handleFetchModels() {
    try {
        await formRef.value.validateFields()
    } catch {
        return
    }

    try {
        fetching.value = true
        const { success, data } = await apis.provider
            .fetchProviderModels(providerId.value, {
                base_url: formData.value.base_url,
                api_key: formData.value.api_key || '',
            })
            .catch(() => {
                throw new Error()
            })
        fetching.value = false
        fetched.value = true
        if (config('http.code.success') === success) {
            modelList.value = data?.models || []
            selectedModels.value = []
        }
    } catch {
        fetching.value = false
        fetched.value = true
    }
}

function filterSpaceOption(input, option) {
    return (
        option.key.toLowerCase().indexOf(input.toLowerCase()) >= 0 ||
        option.value.toLowerCase().indexOf(input.toLowerCase()) >= 0
    )
}

async function loadSpaces() {
    try {
        const { success, data } = await apis.space.getSpaceList({ pageSize: 1000 })
        if (config('http.code.success') === success) {
            spaceOptions.value = data || []
            if (spaceOptions.value.length > 0 && !formData.value.space_code) {
                formData.value.space_code = spaceOptions.value[0].code
            }
        }
    } catch {
        // no-op
    }
}

function handleConfirm() {
    emit('confirm', {
        providerId: providerId.value,
        space_code: formData.value.space_code,
        base_url: formData.value.base_url,
        api_key: formData.value.api_key,
        api_keys: providerApiKeys.value,
        models: selectedModels.value.map((id) => {
            const model = modelList.value.find((m) => m.id === id)
            return {
                id,
                owned_by: model?.owned_by || '',
            }
        }),
    })
    handleClose()
}

function handleOpen(record) {
    providerId.value = record.id
    providerApiKeys.value = Array.isArray(record.api_keys) ? cloneDeep(record.api_keys) : []
    const defaultBaseUrl = record.url || ''
    const defaultApiKey = providerApiKeys.value.length > 0 ? providerApiKeys.value[0] : undefined
    formData.value = {
        space_code: undefined,
        base_url: defaultBaseUrl,
        api_key: defaultApiKey,
    }
    loadSpaces()

    modelList.value = []
    selectedModels.value = []
    searchQuery.value = ''
    fetched.value = false
    visible.value = true
}

function onAfterOpen() {
    // no-op
}

function handleClose() {
    visible.value = false
    searchQuery.value = ''
    formRef.value?.resetFields()
}

defineExpose({
    handleOpen,
})
</script>

<style lang="less" scoped>
.model-list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
}

.model-count {
    color: var(--color-text-tertiary);
    font-size: 13px;
}

.model-id {
    font-family: monospace;
    font-size: 13px;
}

:deep(.ant-form-item-required::before) {
    display: none !important;
}

:deep(.ant-form-item-required::after) {
    display: inline-block;
    margin-left: 4px;
    color: #ff4d4f;
    font-size: 14px;
    font-family: SimSun, sans-serif;
    line-height: 1;
    content: '*';
}
</style>
