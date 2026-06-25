<template>
    <div class="app-page">
        <a-card
            type="flex"
            class="app-card">
            <a-row
                :gutter="16"
                align="middle"
                class="mb-8-2">
                <a-col flex="none">
                    <a-button
                        v-action="'add'"
                        type="primary"
                        ghost
                        @click="$refs.editDialogRef.handleCreate()">
                        <template #icon>
                            <plus-outlined></plus-outlined>
                        </template>
                        {{ $t('pages.provider.add') }}
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            :label="$t('pages.provider.form.code')"
                            name="code"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.provider.form.code.placeholder')"
                                v-model:value="searchFormData.code"
                                style="width: 200px"
                                @pressEnter="handleSearch"></a-input>
                        </a-form-item>
                        <a-form-item
                            :label="$t('pages.provider.form.name')"
                            name="name"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.provider.form.name.placeholder')"
                                v-model:value="searchFormData.name"
                                style="width: 200px"
                                @pressEnter="handleSearch"></a-input>
                        </a-form-item>
                        <a-form-item style="margin-bottom: 0">
                            <x-filter-actions
                                @reset="handleResetSearch"
                                @search="handleSearch" />
                        </a-form-item>
                    </a-form>
                </a-col>
            </a-row>
            <a-table
                :columns="columns"
                :data-source="listData"
                :loading="loading"
                :pagination="paginationState"
                :scroll="{ x: 'max-content' }"
                @change="onTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="'name' === column.key">
                        <a @click="goToDetail(record)">
                            {{ record.name }}
                        </a>
                    </template>
                    <template v-if="'enabled' === column.key">
                        <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                            {{
                                record.enabled === 1
                                    ? $t('pages.provider.form.enabled.active')
                                    : $t('pages.provider.form.enabled.inactive')
                            }}
                        </a-tag>
                    </template>
                    <template v-if="'protocol' === column.key">
                        <a-tag
                            v-if="record.protocol"
                            color="blue"
                            >{{ record.protocol }}</a-tag
                        >
                        <span v-else>--</span>
                    </template>
                    <template v-if="'created_at' === column.key">
                        {{ formatUtcDateTime(record.created_at) }}
                    </template>

                    <template v-if="'action' === column.key">
                        <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                            <a-tooltip>
                                <template #title> {{ $t('pages.provider.edit') }}</template>
                                <edit-outlined />
                            </a-tooltip>
                        </x-action-button>
                        <x-action-button @click="handleRemove(record)">
                            <a-tooltip>
                                <template #title> {{ $t('button.delete') }}</template>
                                <delete-outlined style="color: #ff4d4f" />
                            </a-tooltip>
                        </x-action-button>
                        <x-action-button @click="handleFetchModels(record)">
                            <a-tooltip>
                                <template #title> {{ $t('pages.provider.fetchModels.btn') }}</template>
                                <import-outlined style="color: #1890ff" />
                            </a-tooltip>
                        </x-action-button>
                    </template>
                </template>
            </a-table>
        </a-card>

        <edit-dialog
            ref="editDialogRef"
            @ok="onOk"></edit-dialog>

        <fetch-models-drawer
            ref="fetchModelsDrawerRef"
            @confirm="onFetchModelsConfirm"></fetch-models-drawer>

        <import-mapping-dialog
            ref="importMappingDialogRef"
            @ok="getPageList"></import-mapping-dialog>
    </div>
</template>

<script setup>
import { message, Modal, Radio } from 'ant-design-vue'
import { ref, h } from 'vue'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './ProviderEditDialog.vue'
import FetchModelsDrawer from './ProviderFetchModelsDrawer.vue'
import ImportMappingDialog from './ProviderImportMappingDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, ImportOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

defineOptions({
    name: 'providerList',
})
const router = useRouter()
const { t } = useI18n()

function goToDetail(record) {
    router.push({ name: 'providerDetail', params: { id: record.id } })
}
const columns = [
    {
        title: t('pages.provider.form.name'),
        dataIndex: 'name',
        key: 'name',
        minWidth: 200,
        ellipsis: {
            showTitle: true,
        },
        sorter: (a, b) => (a.name || '').localeCompare(b.name || ''),
    },
    {
        title: t('pages.provider.form.code'),
        dataIndex: 'code',
        minWidth: 180,
        ellipsis: {
            showTitle: true,
        },
        sorter: (a, b) => (a.code || '').localeCompare(b.code || ''),
    },
    { title: t('pages.provider.form.protocol'), dataIndex: 'protocol', key: 'protocol', width: 120 },
    { title: t('pages.provider.form.enabled'), key: 'enabled', width: 100 },
    {
        title: t('pages.provider.form.url'),
        dataIndex: 'url',
        minWidth: 240,
        ellipsis: {
            showTitle: true,
        },
    },
    { title: t('pages.provider.form.creator'), dataIndex: 'creator', width: 120 },
    { title: t('pages.provider.form.description'), dataIndex: 'description', ellipsis: true },
    {
        title: t('pages.provider.form.created_at'),
        key: 'created_at',
        fixed: 'right',
        width: 180,
        sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 180 },
]

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
const editDialogRef = ref()
const fetchModelsDrawerRef = ref()
const importMappingDialogRef = ref()

getPageList()

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.provider
            .getProviderList({
                pageSize,
                current,
                ...searchFormData.value,
            })
            .catch(() => {
                throw new Error()
            })
        hideLoading()
        if (config('http.code.success') === success) {
            listData.value = data
            paginationState.total = total
        }
    } catch (error) {
        hideLoading()
    }
}

function handleRemove({ id }) {
    Modal.confirm({
        title: t('pages.provider.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.provider.delProvider(id).catch((err) => {
                            // Don't show error here, interceptor already shows it
                            // Just check if it's the specific associated error
                            const detail = err?.response?.data?.error?.detail || err?.message || ''
                            if (
                                detail.includes('pages.provider.delAssociatedError') ||
                                detail.includes('Cannot delete provider with associated models')
                            ) {
                                // Show translated message with specific key to avoid duplicate
                                message.error({
                                    content: t('pages.provider.delAssociatedError'),
                                    key: 'DELETE_ERROR',
                                })
                            }
                            reject()
                            return null
                        })

                        if (!result) return // Already handled in catch

                        const { success } = result
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await getPageList()
                        } else {
                            // Don't show error here, interceptor already shows it
                            reject()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    })
}

function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
    getPageList()
}

function handleResetSearch() {
    searchFormData.value = {}
    resetPagination()
    getPageList()
}

function handleSearch() {
    resetPagination()
    getPageList()
}

function handleFetchModels(record) {
    fetchModelsDrawerRef.value.handleOpen(record)
}

async function onFetchModelsConfirm({ providerId, space_code, base_url, api_key, api_keys, models }) {
    if (!models || models.length === 0) return

    const provider = listData.value.find((p) => p.id === providerId)
    const protocol = provider?.protocol || ''

    let keysToCreate = []

    if (Array.isArray(api_keys) && api_keys.length > 1) {
        const importMode = ref('all')
        try {
            await new Promise((resolve, reject) => {
                Modal.confirm({
                    title: t('pages.provider.fetchModels.api_keys_confirm_title', '检测到多个 API 密钥'),
                    width: 600,
                    okText: t('button.confirm', '确认'),
                    cancelText: t('button.cancel', '取消'),
                    content: () => {
                        return h('div', { style: { marginTop: '12px' } }, [
                            h(
                                'p',
                                { style: { marginBottom: '16px', color: 'var(--color-text-secondary)' } },
                                `当前供应商配置了 ${api_keys.length} 个 API 密钥。请选择端点（Endpoint）创建模式：`
                            ),
                            h(
                                Radio.Group,
                                {
                                    value: importMode.value,
                                    'onUpdate:value': (val) => {
                                        importMode.value = val
                                    },
                                },
                                [
                                    h(
                                        Radio,
                                        {
                                            value: 'current',
                                            style: { display: 'block', marginBottom: '8px' },
                                        },
                                        `仅为当前选择/输入的密钥创建端点（每个模型创建 1 个端点）`
                                    ),
                                    h(
                                        Radio,
                                        {
                                            value: 'all',
                                            style: { display: 'block' },
                                        },
                                        `为所有配置的密钥分别创建端点（每个模型创建 ${api_keys.length} 个端点）`
                                    ),
                                ]
                            ),
                        ])
                    },
                    onOk: () => {
                        if (importMode.value === 'current') {
                            keysToCreate = [api_key || '']
                        } else {
                            keysToCreate = api_keys
                        }
                        resolve()
                    },
                    onCancel: () => {
                        reject(new Error('USER_CANCEL'))
                    },
                })
            })
        } catch (err) {
            if (err?.message === 'USER_CANCEL') {
                return
            }
            throw err
        }
    } else {
        if (api_key && (!Array.isArray(api_keys) || !api_keys.includes(api_key))) {
            keysToCreate = [api_key]
        } else if (Array.isArray(api_keys) && api_keys.length > 0) {
            keysToCreate = api_keys
        } else {
            keysToCreate = [api_key || '']
        }
    }

    importMappingDialogRef.value.handleOpen({
        providerId,
        space_code,
        base_url,
        keysToCreate,
        protocol,
        models,
    })
}

async function onOk() {
    await getPageList()
}
</script>
