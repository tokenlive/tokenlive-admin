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
                            {{ $t('pages.tenant.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.tenant.form.code')"
                                name="code"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.tenant.form.code.placeholder')"
                                    v-model:value="searchFormData.code"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.tenant.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.tenant.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item style="margin-bottom: 0">
                                <a-space :size="8">
                                    <a-tooltip :title="$t('common.reset')">
                                        <a-button
                                            shape="circle"
                                            @click="handleResetSearch">
                                            <template #icon>
                                                <redo-outlined />
                                            </template>
                                        </a-button>
                                    </a-tooltip>
                                    <a-tooltip :title="$t('common.search')">
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
                        <!-- 租户名称 -->
                        <template v-if="'name' === column.key">
                            <router-link
                                :to="{ name: 'tenantDetail', params: { id: record.id } }"
                                class="tenant-name-link">
                                {{ record.name }}
                            </router-link>
                        </template>

                        <!-- 租户编码 -->
                        <template v-if="'code' === column.key">
                            <span class="tenant-code-container">
                                <span class="tenant-code">{{ record.code }}</span>
                                <a-tooltip :title="$t('pages.tenant.copy.code')">
                                    <copy-outlined
                                        class="copy-btn-icon"
                                        @click="handleCopyCode(record.code)" />
                                </a-tooltip>
                            </span>
                        </template>

                        <!-- 状态 -->
                        <template v-if="'status' === column.key">
                            <a-tag
                                v-if="record.status === 'activated'"
                                color="success"
                                >{{ $t('pages.tenant.form.status.activated') }}</a-tag
                            >
                            <a-tag
                                v-else
                                color="error"
                                >{{ $t('pages.tenant.form.status.freezed') }}</a-tag
                            >
                        </template>

                        <!-- 创建时间 -->
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <!-- 操作 -->
                        <template v-if="'action' === column.key">
                            <x-action-button @click="editDialogRef.handleEdit(record)">
                                <a-tooltip :title="$t('pages.tenant.edit')">
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleDelete(record)">
                                <a-tooltip :title="$t('pages.tenant.delete')">
                                    <delete-outlined style="color: #ff4d4f" />
                                </a-tooltip>
                            </x-action-button>
                        </template>
                    </template>
                </a-table>
            </a-card>
        </a-col>
    </a-row>

    <!-- 编辑模态框 -->
    <edit-dialog
        ref="editDialogRef"
        @ok="onOk"></edit-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { useI18n } from 'vue-i18n'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './components/EditDialog.vue'
import {
    PlusOutlined,
    EditOutlined,
    DeleteOutlined,
    CopyOutlined,
    SearchOutlined,
    RedoOutlined,
} from '@ant-design/icons-vue'

const { t } = useI18n()

defineOptions({
    name: 'systemTenant',
})

// 表格定义
const columns = computed(() => [
    { title: t('pages.tenant.form.name'), dataIndex: 'name', key: 'name', width: 150 },
    { title: t('pages.tenant.form.code'), dataIndex: 'code', key: 'code', width: 150 },
    { title: t('pages.tenant.form.status'), dataIndex: 'status', key: 'status', width: 100 },
    { title: t('pages.tenant.form.description'), dataIndex: 'description', width: 250 },
    { title: t('pages.tenant.form.created_at'), dataIndex: 'created_at', key: 'created_at', width: 150 },
    { title: t('common.action'), key: 'action', fixed: 'right', width: 90 },
])

const { listData, loading, showLoading, hideLoading, paginationState, resetPagination, searchFormData } =
    usePagination()

const editDialogRef = ref()

/**
 * 获取租户列表
 */
async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState

        // 传递查询参数
        const params = {
            pageSize,
            current,
            code: searchFormData.value.code || undefined,
            name: searchFormData.value.name || undefined,
        }

        const { success, data, total } = await apis.tenant.getList(params).catch(() => {
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
 * 删除租户
 */
function handleDelete({ id, name, code }) {
    Modal.confirm({
        title: t('pages.tenant.delete.title'),
        content: t('pages.tenant.delete.content', { name, code }),
        okText: t('common.confirm'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.tenant.del(id).catch((err) => {
                            throw err
                        })
                        if (config('http.code.success') === result?.success) {
                            resolve()
                            message.success(t('pages.tenant.delete.success'))
                            await getPageList()
                        } else {
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

/**
 * 分页切换
 */
function onTableChange({ current, pageSize }) {
    paginationState.current = current
    paginationState.pageSize = pageSize
    getPageList()
}

/**
 * 触发搜索
 */
function handleSearch() {
    resetPagination()
    getPageList()
}

/**
 * 重置搜索条件
 */
function handleResetSearch() {
    searchFormData.value = {}
    resetPagination()
    getPageList()
}

/**
 * 保存成功回调
 */
async function onOk() {
    await getPageList()
}

/**
 * 一键复制租户编码
 */
function handleCopyCode(code) {
    if (navigator.clipboard) {
        navigator.clipboard
            .writeText(code)
            .then(() => message.success(t('pages.tenant.copy.success')))
            .catch(() => message.error(t('common.copy.failed')))
    } else {
        const input = document.createElement('input')
        input.setAttribute('value', code)
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        document.body.removeChild(input)
        message.success(t('pages.tenant.copy.success'))
    }
}

// 初始化加载数据
;(async () => {
    await new Promise((resolve) => setTimeout(resolve, 100))
    await getPageList()
})()
</script>

<style lang="less" scoped>
// 搜索栏行内间距
:deep(.ant-form-inline) {
    .ant-form-item {
        margin-right: 16px;

        &:last-child {
            margin-right: 0;
        }
    }
}

.tenant-code-container {
    display: inline-flex;
    align-items: center;
    gap: 8px;

    .copy-btn-icon {
        color: var(--color-primary);
        cursor: pointer;
        font-size: 14px;
        transition:
            color 0.3s ease,
            transform 0.2s ease;

        &:hover {
            color: var(--color-primary-hover);
            transform: scale(1.15);
        }
        &:active {
            transform: scale(0.95);
        }
    }
}

.tenant-code {
    font-family: Menlo, Monaco, Consolas, monospace;
    font-size: 13px;
    background: var(--color-bg-active);
    padding: 2px 6px;
    border-radius: 4px;
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
}

.tenant-name-link {
    font-weight: 500;
    color: var(--color-primary);
    transition: color 0.3s ease;

    &:hover {
        color: var(--color-primary-hover);
    }
}

// 操作按钮悬停优化
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
