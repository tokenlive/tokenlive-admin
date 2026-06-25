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
                            {{ $t('pages.user-api-key.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.user-api-key.form.user')"
                                name="user_id"
                                style="margin-bottom: 0">
                                <a-select
                                    v-model:value="searchFormData.user_id"
                                    show-search
                                    :placeholder="$t('pages.user-api-key.form.user.placeholder')"
                                    :filter-option="filterUserOption"
                                    style="width: 200px"
                                    allow-clear>
                                    <a-select-option
                                        v-for="item in userOptions"
                                        :key="item.value"
                                        :value="item.value"
                                        :label="item.label">
                                        {{ item.label }}
                                    </a-select-option>
                                </a-select>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.user-api-key.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.user-api-key.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 180px"
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
                    :scroll="{ x: 1100 }"
                    @change="onTableChange">
                    <template #bodyCell="{ column, record }">
                        <!-- 关联用户 -->
                        <template v-if="'user' === column.key">
                            <span class="user-badge">
                                {{ userMap[String(record.user_id)]?.label || record.user_id }}
                            </span>
                        </template>

                        <!-- API Key -->
                        <template v-if="'api_key' === column.key">
                            <span class="api-key-container">
                                <span class="api-key-code">{{ record.api_key }}</span>
                                <a-tooltip :title="$t('pages.user-api-key.copy')">
                                    <copy-outlined
                                        class="copy-btn-icon"
                                        @click="handleCopyKey(record.id)" />
                                </a-tooltip>
                            </span>
                        </template>

                        <!-- 状态 -->
                        <template v-if="'status' === column.key">
                            <a-tag
                                v-if="record.status === 1"
                                color="success"
                                >{{ $t('pages.user-api-key.form.status.enabled') }}</a-tag
                            >
                            <a-tag
                                v-else
                                color="error"
                                >{{ $t('pages.user-api-key.form.status.disabled') }}</a-tag
                            >
                        </template>

                        <!-- 额度 -->
                        <template v-if="'quota' === column.key">
                            <a-tag
                                v-if="record.quota === -1"
                                color="blue"
                                >{{ $t('pages.user-api-key.form.quota.unlimited') }}</a-tag
                            >
                            <span
                                v-else
                                class="quota-value"
                                >{{ record.quota.toLocaleString() }} Tokens</span
                            >
                        </template>

                        <!-- 过期时间 -->
                        <template v-if="'expires_at' === column.key">
                            <a-tag
                                v-if="!record.expires_at"
                                color="cyan"
                                >{{ $t('pages.user-api-key.form.expires_at.never') }}</a-tag
                            >
                            <span
                                v-else
                                :class="{ expired: isExpired(record.expires_at) }">
                                {{ formatUtcDateTime(record.expires_at) }}
                            </span>
                        </template>

                        <!-- 创建时间 -->
                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <!-- 操作 -->
                        <template v-if="'action' === column.key">
                            <x-action-button @click="editDialogRef.handleEdit(record)">
                                <a-tooltip :title="$t('pages.user-api-key.edit')">
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="handleDelete(record)">
                                <a-tooltip :title="$t('pages.user-api-key.delete')">
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
import { ref, computed, watchEffect } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { useI18n } from 'vue-i18n'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './components/EditDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'

const { t } = useI18n()

defineOptions({
    name: 'systemUserAPIKey',
})

// 表格定义
const columns = computed(() => [
    { title: t('pages.user-api-key.form.name'), dataIndex: 'name', width: 140 },
    { title: t('pages.user-api-key.form.user'), dataIndex: 'user_id', key: 'user', width: 150 },
    { title: t('pages.user-api-key.form.api_key'), dataIndex: 'api_key', key: 'api_key', width: 160 },
    { title: t('pages.user-api-key.form.status'), dataIndex: 'status', key: 'status', width: 70 },
    { title: t('pages.user-api-key.form.quota'), dataIndex: 'quota', key: 'quota', width: 130 },
    { title: t('pages.user-api-key.form.expires_at'), dataIndex: 'expires_at', key: 'expires_at', width: 150 },
    { title: t('pages.user-api-key.form.created_at'), dataIndex: 'created_at', key: 'created_at', width: 150 },
    { title: t('common.action'), key: 'action', fixed: 'right', width: 90 },
])

const { listData, loading, showLoading, hideLoading, paginationState, resetPagination, searchFormData } =
    usePagination()

const editDialogRef = ref()
const userOptions = ref([])
const userMap = ref({})

watchEffect(() => {
    const superAdmin = userOptions.value.find((item) => item.value === 'root')
    if (superAdmin) {
        superAdmin.label = t('pages.user-api-key.super_admin')
    }
})

/**
 * 提前缓存所有用户，用于在列表页进行 ID->姓名 本地映射翻译
 */
async function loadUsersCache() {
    try {
        const result = await apis.users.getUsersList({ pageSize: 100, current: 1 }).catch(() => null)

        const list = [
            {
                label: t('pages.user-api-key.super_admin'),
                value: 'root',
            },
        ]

        if (result && config('http.code.success') === result.success && result.data) {
            list.push(
                ...result.data.map((user) => ({
                    label: `${user.name} (${user.username})`,
                    value: String(user.id),
                }))
            )
        }

        userOptions.value = list

        // 构建 MAP（key 统一为字符串）
        const map = {}
        list.forEach((item) => {
            map[item.value] = item
        })
        userMap.value = map
    } catch (e) {
        // 静默容错
    }
}

/**
 * 用户下拉搜索过滤
 */
function filterUserOption(input, option) {
    const label = option.label || ''
    return label.toLowerCase().includes(input.toLowerCase())
}

/**
 * 判断是否过期
 */
function isExpired(timeStr) {
    if (!timeStr) return false
    return dayjs(timeStr).isBefore(dayjs())
}

/**
 * 获取 API Key 列表
 */
async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState

        // 传递查询参数
        const params = {
            pageSize,
            current,
            user_id: searchFormData.value.user_id || undefined,
            name: searchFormData.value.name || undefined,
        }

        const { success, data, total } = await apis.user_api_key.getList(params).catch(() => {
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
 * 删除 API Key
 */
function handleDelete({ id, name }) {
    Modal.confirm({
        title: t('pages.user-api-key.delete.title'),
        content: t('pages.user-api-key.delete.content', { name }),
        okText: t('common.confirm'),
        okType: 'danger',
        cancelText: t('common.cancel'),
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.user_api_key.del(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('pages.user-api-key.delete.success'))
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
 * 一键复制 API Key 明文到剪贴板
 */
async function handleCopyKey(id) {
    try {
        const { success, data } = await apis.user_api_key.getPlaintext(id).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') !== success || !data) {
            message.error(t('pages.user-api-key.plaintext.failed'))
            return
        }

        if (navigator.clipboard) {
            navigator.clipboard
                .writeText(data)
                .then(() => message.success(t('pages.user-api-key.copy.success')))
                .catch(() => message.error(t('pages.user-api-key.copy.failed')))
        } else {
            const input = document.createElement('input')
            input.setAttribute('value', data)
            document.body.appendChild(input)
            input.select()
            document.execCommand('copy')
            document.body.removeChild(input)
            message.success(t('pages.user-api-key.copy.success'))
        }
    } catch (error) {
        message.error(t('pages.user-api-key.copy.retry'))
    }
}

// 初始化加载缓存并查询数据
;(async () => {
    // 延迟加载用户缓存，避免 AbortController 取消请求
    await new Promise((resolve) => setTimeout(resolve, 100))
    await loadUsersCache()
    await getPageList()
})()
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

.api-key-container {
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

.api-key-code {
    font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
    font-size: 13px;
    background: var(--color-bg-active);
    padding: 2px 6px;
    border-radius: 4px;
    border: 1px solid var(--color-border);
    color: var(--color-text-primary);
}

.user-badge {
    font-weight: 500;
    color: var(--color-text-primary);
}

.quota-value {
    font-weight: 500;
    color: var(--color-text-primary);
}

.expired {
    color: var(--color-error);
    text-decoration: line-through;
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
    margin-bottom: 16px;
}
</style>
