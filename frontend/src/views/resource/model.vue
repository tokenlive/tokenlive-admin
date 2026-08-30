<template>
    <div
        class="app-page"
        :style="{ height: appStore.mainHeight }">
        <a-card
            type="flex"
            class="app-card app-card--fill">
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
                        {{ $t('pages.model.add') }}
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            :label="$t('pages.model.form.space_code')"
                            name="space_code"
                            style="margin-bottom: 0">
                            <a-select
                                :placeholder="$t('pages.model.form.space_code.placeholder')"
                                v-model:value="searchFormData.space_code"
                                show-search
                                :filter-option="filterSpaceOption"
                                @change="onSpaceChange"
                                style="width: 200px">
                                <a-select-option
                                    v-for="item in spaceOptions"
                                    :key="item.code"
                                    :value="item.code">
                                    {{ item.name }} ({{ item.code }})
                                </a-select-option>
                            </a-select>
                        </a-form-item>
                        <a-form-item
                            :label="$t('pages.model.form.model_name')"
                            name="model_name"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.model.form.model_name.placeholder')"
                                v-model:value="searchFormData.model_name"
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
            <div
                ref="tableContainerRef"
                class="table-fill-region"
                :style="tableContainerStyle">
                <a-table
                    :columns="columns"
                    :data-source="listData"
                    :loading="loading"
                    :pagination="paginationState"
                    :scroll="{ x: 'max-content', y: tableScrollY || undefined }"
                    @change="onTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'model_name' === column.key">
                            <a @click="goToDetail(record)">
                                {{ record.model_name }}
                            </a>
                        </template>
                        <template v-if="'model_code' === column.key">
                            {{ record.model_code }}
                            <a-tooltip :title="$t('button.copy')">
                                <copy-outlined
                                    style="margin-left: 6px; color: #999; cursor: pointer; font-size: 12px"
                                    @click="handleCopy(record.model_code)" />
                            </a-tooltip>
                        </template>
                        <template v-if="'enabled' === column.key">
                            <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                {{
                                    record.enabled === 1
                                        ? $t('pages.model.form.enabled.active')
                                        : $t('pages.model.form.enabled.inactive')
                                }}
                            </a-tag>
                        </template>
                        <template v-if="'recent_status' === column.key">
                            <EndpointStatusStrip :points="record.status_points || []" />
                        </template>
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.model.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleToggleEnabled(record)">
                                <a-tooltip>
                                    <template #title>{{
                                        record.enabled === 1
                                            ? $t('pages.endpoint.disable')
                                            : $t('pages.endpoint.enable')
                                    }}</template>
                                    <poweroff-outlined :style="{ color: record.enabled === 1 ? '#faad14' : '#52c41a' }"
                                /></a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleSync(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.model.sync') }}</template>
                                    <sync-outlined style="color: #1890ff" />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemove(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('button.delete') }}</template>
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </div>
        </a-card>

        <edit-dialog
            ref="editDialogRef"
            :space-options="spaceOptions"
            @ok="onOk"></edit-dialog>
    </div>
</template>

<script setup>
import { message, Modal } from 'ant-design-vue'
import { ref, onMounted, onUnmounted } from 'vue'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination, useTableAutoScrollY } from '@/hooks'
import { useAppStore } from '@/store'
import EditDialog from './ModelEditDialog.vue'
import EndpointStatusStrip from '@/components/EndpointStatusStrip.vue'
import {
    PlusOutlined,
    EditOutlined,
    DeleteOutlined,
    CopyOutlined,
    SyncOutlined,
    PoweroffOutlined,
} from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { initSpaceCode, setCurrentSpaceCode } from '@/utils/spaceStorage'

defineOptions({
    name: 'modelList',
})
const router = useRouter()
const { t } = useI18n()
const columns = [
    {
        title: t('pages.model.form.model_name'),
        dataIndex: 'model_name',
        key: 'model_name',
        minWidth: 200,
        ellipsis: {
            showTitle: true,
        },
        sorter: (a, b) => (a.model_name || '').localeCompare(b.model_name || ''),
        fixed: 'left',
    },
    {
        title: t('pages.model.form.model_code'),
        dataIndex: 'model_code',
        key: 'model_code',
        minWidth: 250,
        ellipsis: {
            showTitle: true,
        },
        sorter: (a, b) => (a.model_code || '').localeCompare(b.model_code || ''),
    },
    { title: t('pages.model.recent_status'), key: 'recent_status', width: 180 },
    { title: t('pages.model.form.owner'), dataIndex: 'owner', width: 120 },
    { title: t('pages.model.form.enabled'), key: 'enabled', width: 80 },
    { title: t('pages.model.form.creator'), dataIndex: 'creator', width: 120 },
    { title: t('pages.model.form.description'), dataIndex: 'description', ellipsis: true },
    {
        title: t('pages.model.form.created_at'),
        key: 'created_at',
        width: 180,
        sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 190 },
]

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
const appStore = useAppStore()
const {
    scrollY: tableScrollY,
    containerRef: tableContainerRef,
    containerStyle: tableContainerStyle,
} = useTableAutoScrollY()
const editDialogRef = ref()
const spaceOptions = ref([])

loadSpaceOptions()

async function loadSpaceOptions() {
    try {
        const { success, data } = await apis.space.getSpaceList({ pageSize: 99, current: 1 }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            spaceOptions.value = data || []
            if (spaceOptions.value.length > 0) {
                searchFormData.value.space_code = initSpaceCode(spaceOptions.value)
                getPageList()
            }
        }
    } catch (error) {
        // ignore
    }
}

function onSpaceChange(value) {
    setCurrentSpaceCode(value)
    resetPagination()
    getPageList()
}

function filterSpaceOption(input, option) {
    const label = option.children?.[0]?.children || ''
    return option.value.toLowerCase().includes(input.toLowerCase()) || label.toLowerCase().includes(input.toLowerCase())
}

async function getPageList(isSilent = false) {
    try {
        if (!isSilent) showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.model
            .getModelList({
                pageSize,
                current,
                ...searchFormData.value,
            })
            .catch(() => {
                throw new Error()
            })
        if (!isSilent) hideLoading()
        if (config('http.code.success') === success) {
            listData.value = data
            paginationState.total = total
        }
    } catch (error) {
        if (!isSilent) hideLoading()
    }
}

function handleRemove({ id, model_name }) {
    const showDeleteConfirm = () => {
        const getRequestErrorMessage = (error) =>
            error?.response?.data?.error?.detail || error?.message || t('component.message.error.request')

        Modal.confirm({
            title: t('pages.model.delTip'),
            okText: t('button.confirm'),
            okType: 'danger',
            onOk: async () => {
                try {
                    const { success } = await apis.model.delModel(id)
                    if (config('http.code.success') === success) {
                        message.success(t('component.message.success.delete'))
                        await getPageList()
                    }
                } catch (error) {
                    message.error(getRequestErrorMessage(error))
                    throw error
                }
            },
        })
    }

    // 删除前只做 endpoint 提示；后端仍是最终一致性校验来源
    apis.endpoint
        .getEndpointsByModelId(id)
        .then(({ success, data }) => {
            if (config('http.code.success') === success && data && data.length > 0) {
                Modal.warning({
                    title: t('pages.model.delEndpointTip'),
                    content: `${model_name} - ${data.length} 个Endpoint`,
                    okText: t('button.confirm'),
                })
                return
            }
            showDeleteConfirm()
        })
        .catch(() => {
            // 查询失败时仍然允许删除
            showDeleteConfirm()
        })
}

function handleSync({ id, model_name }) {
    Modal.confirm({
        title: t('pages.model.syncTip', { name: model_name }),
        content: t('pages.model.syncContent'),
        okText: t('button.confirm'),
        cancelText: t('button.cancel'),
        onOk: async () => {
            try {
                const { success } = await apis.model.syncModel(id)
                if (config('http.code.success') === success) {
                    message.success(t('component.message.success.operation'))
                } else {
                    message.error(t('component.message.error.operation'))
                }
            } catch (error) {
                message.error(t('component.message.error.request'))
            }
        },
    })
}

const togglingModels = ref({})

async function handleToggleEnabled(record) {
    if (togglingModels.value[record.id]) return
    const nextEnabled = record.enabled === 1 ? 0 : 1
    togglingModels.value[record.id] = true
    try {
        const { success } = await apis.model.toggleModelEnabled(record.id, { enabled: nextEnabled }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            message.success(
                nextEnabled === 1 ? t('pages.endpoint.enable.success') : t('pages.endpoint.disable.success')
            )
            await getPageList()
        }
    } catch (error) {
        // ignore, error already handled by interceptor
    } finally {
        togglingModels.value[record.id] = false
    }
}

function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
    getPageList()
}

function handleResetSearch() {
    const spaceCode = searchFormData.value.space_code
    searchFormData.value = { space_code: spaceCode }
    resetPagination()
    getPageList()
}

function handleSearch() {
    resetPagination()
    getPageList()
}

function goToDetail(record) {
    router.push({ name: 'modelDetail', params: { id: record.id } })
}

function handleCopy(text) {
    if (navigator.clipboard) {
        navigator.clipboard
            .writeText(text)
            .then(() => {
                message.success(t('component.message.success.copy'))
            })
            .catch(() => {
                message.error(t('component.message.error.copy'))
            })
    } else {
        const input = document.createElement('input')
        input.setAttribute('value', text)
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
        message.success(t('component.message.success.copy'))
    }
}

async function onOk() {
    await getPageList()
}

let timer = null
onMounted(() => {
    timer = setInterval(() => {
        getPageList(true)
    }, 10000)
})

onUnmounted(() => {
    if (timer) {
        clearInterval(timer)
        timer = null
    }
})
</script>

<style lang="less" scoped>
@import '@/styles/variables.less';

// 模型名称链接 - 添加平滑过渡
:deep(.ant-table-tbody) {
    a {
        color: @color-primary;
        transition: color 0.2s ease;

        &:hover {
            color: #0958d9;
        }
    }
}
</style>
