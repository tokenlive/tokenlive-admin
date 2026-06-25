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
                            type="primary"
                            ghost
                            @click="editDialogRef.handleCreate()">
                            <template #icon>
                                <plus-outlined></plus-outlined>
                            </template>
                            {{ $t('pages.system.user.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.system.user.form.username')"
                                name="username"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.user.form.username.placeholder')"
                                    v-model:value="searchFormData.username"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.system.user.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.user.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item style="margin-bottom: 0">
                                <x-filter-actions
                                    @reset="handleResetSearch"
                                    @search="handleSearch" />
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
                        <!-- 所属租户 -->
                        <template v-if="'tenant' === column.key">
                            <span class="tenant-badge">
                                {{ tenantMap[record.tenant]?.name || record.tenant || '-' }}
                            </span>
                        </template>

                        <!-- 状态 -->
                        <template v-if="'statusType' === column.key">
                            <a-tag
                                v-if="statusUserTypeEnum.is('activated', record.status)"
                                color="success">
                                {{ statusUserTypeEnum.getDesc(record.status) }}
                            </a-tag>
                            <a-tag
                                v-if="statusUserTypeEnum.is('freezed', record.status)"
                                color="error">
                                {{ statusUserTypeEnum.getDesc(record.status) }}
                            </a-tag>
                        </template>

                        <!-- 创建时间 -->
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <!-- 操作 -->
                        <template v-if="'action' === column.key">
                            <x-action-button @click="editDialogRef.handleEdit(record)">
                                <a-tooltip :title="$t('pages.system.user.edit')">
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleDelete(record)">
                                <a-tooltip :title="$t('pages.system.delete')">
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
import { computed, ref } from 'vue'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { statusUserTypeEnum } from '@/enums/system'
import { usePagination } from '@/hooks'

import EditDialog from './components/EditDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'systemUser',
})

const { t } = useI18n()
const columns = computed(() => [
    { title: t('pages.system.user.form.username'), dataIndex: 'username', width: 120 },
    { title: t('pages.system.user.form.name'), dataIndex: 'name', key: 'name', width: 100 },
    { title: t('pages.system.user.form.tenant'), dataIndex: 'tenant', key: 'tenant', width: 120 },
    { title: t('pages.system.user.form.phone'), dataIndex: 'phone', width: 120 },
    { title: t('pages.system.user.form.email'), dataIndex: 'email', width: 140 },
    { title: t('pages.system.user.form.status'), dataIndex: 'status', key: 'statusType', width: 80 },
    { title: t('pages.system.user.form.created_at'), key: 'created_at', fixed: 'right', width: 150 },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 90 },
])

const { listData, loading, showLoading, hideLoading, paginationState, resetPagination, searchFormData } =
    usePagination()

const editDialogRef = ref()
const tenantMap = ref({})

/**
 * 缓存加载所有租户，用于名称翻译
 */
async function loadTenantsCache() {
    try {
        const result = await apis.tenant.getList({ pageSize: 100, current: 1 }).catch(() => null)
        const map = {}
        if (result && config('http.code.success') === result.success && result.data) {
            result.data.forEach((t) => {
                map[t.code] = t
            })
        }
        tenantMap.value = map
    } catch (e) {
        // 静默容错
    }
}

/**
 * 获取用户列表
 */
async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.users
            .getUsersList({
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
 * 删除
 */
function handleDelete({ id }) {
    Modal.confirm({
        title: t('pages.system.user.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.users.delUsers(id).catch(() => {
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
 * 分页切换
 */
function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
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
 * 重置
 */
function handleResetSearch() {
    searchFormData.value = {}
    resetPagination()
    getPageList()
}

/**
 * 编辑完成回调
 */
async function onOk() {
    await getPageList()
}

// 延迟加载缓存并数据查询
;(async () => {
    await new Promise((resolve) => setTimeout(resolve, 100))
    await loadTenantsCache()
    await getPageList()
})()
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

.tenant-badge {
    font-weight: 500;
    color: #1890ff;
    background: #e6f7ff;
    padding: 2px 6px;
    border-radius: 4px;
    border: 1px solid #91d5ff;
}

// 按钮悬停优化
:deep(.x-action-button) {
    transition: all 0.2s ease;

    &:hover {
        background-color: var(--color-bg-hover);
        border-radius: 4px;
    }
}

:deep(.ant-table) {
    .ant-table-tbody > tr > td {
        padding: 12px 16px;
    }

    .ant-table-thead > tr > th {
        padding: 12px 16px;
        font-weight: 600;
    }
}

.mb-8-2 {
    padding-bottom: 16px;
    margin-bottom: 16px;
}
</style>
