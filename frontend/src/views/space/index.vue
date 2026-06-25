<template>
    <a-row
        :gutter="8"
        :wrap="false">
        <a-col flex="auto">
            <a-card type="flex">
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
                            {{ $t('pages.space.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.space.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.space.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 200px"
                                    @pressEnter="handleSearch"></a-input>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.space.form.code')"
                                name="code"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.space.form.code.placeholder')"
                                    v-model:value="searchFormData.code"
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
                    :scroll="{ x: 1000 }"
                    @change="onTableChange">
                    <template #bodyCell="{ column, record }">
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.space.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemove(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.system.delete') }}</template>
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </a-card>
        </a-col>
    </a-row>

    <edit-dialog
        ref="editDialogRef"
        @ok="onOk"></edit-dialog>
</template>

<script setup>
import { message, Modal } from 'ant-design-vue'
import { ref } from 'vue'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './components/EditDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'spaceList',
})
const { t } = useI18n()
const columns = [
    { title: t('pages.space.form.code'), dataIndex: 'code', width: 200 },
    { title: t('pages.space.form.name'), dataIndex: 'name', width: 200 },
    { title: t('pages.space.form.tenant'), dataIndex: 'tenant', width: 150 },
    { title: t('pages.space.form.creator'), dataIndex: 'creator', width: 120 },
    { title: t('pages.space.form.description'), dataIndex: 'description', ellipsis: true },
    { title: t('pages.space.form.created_at'), key: 'created_at', fixed: 'right', width: 180 },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 120 },
]

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
const editDialogRef = ref()

getPageList()

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.space
            .getSpaceList({
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
        title: t('pages.space.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.space.delSpace(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await getPageList()
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

async function onOk() {
    await getPageList()
}
</script>

<style lang="less" scoped>
@import '@/styles/variables.less';

// 搜索栏和操作按钮行
:deep(.ant-form-inline) {
    .ant-form-item {
        margin-right: 16px;

        &:last-child {
            margin-right: 0;
        }
    }
}

// 操作按钮 - 添加悬停效果
:deep(.x-action-button) {
    transition: all 0.2s ease;

    &:hover {
        background-color: var(--color-bg-hover);
        border-radius: 4px;
    }
}

// 搜索按钮 - 添加轻微过渡
:deep(.ant-btn) {
    transition: all 0.2s ease;
}

// 表格单元格 - 优化间距
:deep(.ant-table) {
    .ant-table-tbody > tr > td {
        padding: 12px 16px;
    }

    .ant-table-thead > tr > th {
        padding: 12px 16px;
        font-weight: 600;
    }
}

// 搜索栏分隔线
.mb-8-2 {
    padding-bottom: 16px;
    /* removed border-bottom for dark mode */
    margin-bottom: 16px;
}
</style>
