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
                            {{ $t('pages.system.menu.add') }}
                        </a-button>
                    </a-col>
                    <a-col flex="auto"></a-col>
                    <a-col flex="none">
                        <a-form
                            :model="searchFormData"
                            layout="inline">
                            <a-form-item
                                :label="$t('pages.system.menu.form.name')"
                                name="name"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.menu.form.name.placeholder')"
                                    v-model:value="searchFormData.name"
                                    style="width: 160px"
                                    @pressEnter="handleSearch"
                                    allow-clear></a-input>
                            </a-form-item>
                            <a-form-item
                                :label="$t('pages.system.menu.form.code')"
                                name="code"
                                style="margin-bottom: 0">
                                <a-input
                                    :placeholder="$t('pages.system.menu.form.code.placeholder')"
                                    v-model:value="searchFormData.code"
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
                    :pagination="true"
                    :scroll="{ x: 1000 }">
                    <template #bodyCell="{ column, record }">
                        <!-- 菜单类型 -->
                        <template v-if="'menuType' === column.key">
                            <a-tag
                                v-if="menuTypeEnum.is('page', record.type)"
                                color="processing">
                                {{ menuTypeEnum.getDesc(record.type) }}
                            </a-tag>
                            <a-tag
                                v-if="menuTypeEnum.is('button', record.type)"
                                color="success">
                                {{ menuTypeEnum.getDesc(record.type) }}
                            </a-tag>
                        </template>

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
                                <a-tooltip :title="$t('pages.system.menu.edit')">
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>

                            <x-action-button @click="editDialogRef.handleCreateChild(record)">
                                <a-tooltip :title="$t('pages.system.menu.button.addChild')">
                                    <plus-circle-outlined />
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
import { Modal, message } from 'ant-design-vue'
import { ref } from 'vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlusCircleOutlined } from '@ant-design/icons-vue'
import apis from '@/apis'
import { config } from '@/config'
import { menuTypeEnum, statusTypeEnum } from '@/enums/system'
import { usePagination } from '@/hooks'
import { formatUtcDateTime } from '@/utils/util'
import EditDialog from './components/EditDialog.vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'systemMenu',
})

const { t } = useI18n()
const columns = ref([
    { title: t('pages.system.menu.form.name'), dataIndex: 'name', key: 'name', fixed: true },
    { title: t('pages.system.menu.form.code'), dataIndex: 'code', key: 'code' },
    { title: t('pages.system.menu.form.type'), dataIndex: 'type', key: 'menuType', width: 100 },
    { title: t('pages.system.menu.form.status'), dataIndex: 'status', key: 'statusType', width: 100 },
    { title: t('pages.system.menu.form.sequence'), dataIndex: 'sequence', width: 100 },
    { title: t('pages.system.menu.form.created_at'), dataIndex: 'created_at', key: 'created_at', width: 180 },
    { title: t('button.action'), key: 'action', width: 160 },
])

const { listData, loading, showLoading, hideLoading, searchFormData, resetPagination } = usePagination()

const editDialogRef = ref()

getMenuList()

/**
 * 获取菜单列表
 */
async function getMenuList() {
    try {
        showLoading()
        const { data, success } = await apis.menu
            .getMenuList({
                ...searchFormData.value,
            })
            .catch(() => {
                throw new Error()
            })
        hideLoading()
        if (config('http.code.success') === success) {
            data.forEach((item) => {
                item.name = t(item.code) || item.name
            })
            listData.value = data
        }
    } catch (error) {
        hideLoading()
    }
}

/**
 * 搜索
 */
function handleSearch() {
    resetPagination()
    getMenuList()
}

/**
 * 重置
 */
function handleResetSearch() {
    searchFormData.value = {}
    resetPagination()
    getMenuList()
}

/**
 * 删除
 */
function handleDelete({ id }) {
    Modal.confirm({
        title: t('pages.system.menu.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.menu.delMenu(id).catch(() => {
                            throw new Error()
                        })
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await getMenuList()
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
 * 编辑完成回调
 */
async function onOk() {
    message.success(t('component.message.success.delete'))
    await getMenuList()
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
