<template>
    <!-- 数据表格卡片 -->
    <a-row
        :gutter="8"
        :wrap="false">
        <a-col flex="auto">
            <a-card type="flex">
                <!-- 头部：操作按钮 + 搜索栏 -->
                <a-row
                    :gutter="16"
                    align="middle"
                    class="mb-8-2">
                    <a-col flex="none">
                        <a-button
                            v-action="'add'"
                            type="primary"
                            ghost
                            @click="editDialogRef.handleCreate()">
                            <template #icon>
                                <plus-outlined></plus-outlined>
                            </template>
                            {{ $t('pages.system.role.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.system.role.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.role.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.system.role.form.code')"
                                name="code"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.role.form.code.placeholder')"
                                    v-model:value="searchFormData.code"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item style="margin-bottom: 0">
                                <a-space :size="8">
                                    <a-tooltip :title="$t('button.reset')">
                                        <a-button
                                            shape="circle"
                                            @click="handleResetSearch">
                                            <template #icon>
                                                <redo-outlined />
                                            </template>
                                        </a-button>
                                    </a-tooltip>
                                    <a-tooltip :title="$t('button.search')">
                                        <a-button
                                            type="primary"
                                            ghost
                                            shape="circle"
                                            @click="handleSearch">
                                            <template #icon>
                                                <search-outlined />
                                            </template>
                                        </a-button>
                                    </a-tooltip>
                                </a-space>
                            </a-form-item>
                        </a-form>
                    </a-col>
                </a-row>

                <!-- 表格 -->
                <a-table
                    row-key="id"
                    :columns="columns"
                    :data-source="listData"
                    :loading="loading"
                    :pagination="paginationState"
                    :scroll="{ x: 1000 }"
                    @change="onTableChange">
                    <template #bodyCell="{ column, record }">
                        <!-- 状态 -->
                        <template v-if="'statusType' === column.key">
                            <a-tag
                                v-if="statusTypeEnum.is('enabled', record.status)"
                                color="success">
                                {{ statusTypeEnum.getDesc(record.status) }}
                            </a-tag>
                            <a-tag
                                v-else
                                color="error">
                                {{ statusTypeEnum.getDesc(record.status) }}
                            </a-tag>
                        </template>

                        <!-- 创建时间 -->
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <!-- 操作 -->
                        <template v-if="'action' === column.key">
                            <x-action-button @click="editDialogRef.handleEdit(record)">
                                <a-tooltip :title="$t('pages.system.role.edit')">
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleRemove(record)">
                                <a-tooltip :title="$t('pages.system.delete')">
                                    <delete-outlined style="color: var(--color-error)" />
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
import { statusTypeEnum } from '@/enums/system'
import { usePagination } from '@/hooks'
import EditDialog from './components/EditDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, RedoOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'systemRole',
})

const { t } = useI18n()
const columns = [
    { title: t('pages.system.role.form.code'), dataIndex: 'code', width: 200 },
    { title: t('pages.system.role.form.name'), dataIndex: 'name', width: 200 },
    { title: t('pages.system.role.form.status'), dataIndex: 'status', key: 'statusType', width: 100 },
    { title: t('pages.system.role.form.sequence'), dataIndex: 'sequence', width: 100 },
    { title: t('pages.system.role.form.created_at'), key: 'created_at', fixed: 'right', width: 150 },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 140 },
]

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()

const editDialogRef = ref()

getPageList()

/**
 * 获取角色列表
 */
async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.role
            .getRoleList({
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

/**
 * 移除角色
 */
function handleRemove({ id }) {
    Modal.confirm({
        title: t('pages.system.role.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.role.delRole(id).catch(() => {
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

/**
 * 分页
 */
function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
    getPageList()
}

/**
 * 重置
 */
function handleResetSearch() {
    searchFormData.value = {}
    resetPagination()
    getPageList()
}

/**
 * 搜索
 */
function handleSearch() {
    resetPagination()
    getPageList()
}

/**
 * 编辑完成回调
 */
async function onOk() {
    await getPageList()
}
</script>

<style lang="less" scoped>
// 行内搜索表单紧凑间距
:deep(.ant-form-inline) {
    .ant-form-item {
        margin-right: 16px;

        &:last-child {
            margin-right: 0;
        }
    }
}

// 按钮悬停优化
:deep(.x-action-button) {
    transition: all 0.2s ease;

    &:hover {
        background-color: var(--color-bg-hover);
        border-radius: 4px;
    }
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

// 头部间距
.mb-8-2 {
    padding-bottom: 16px;
    margin-bottom: 16px;
}
</style>
