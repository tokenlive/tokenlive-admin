<template>
    <div
        class="app-page"
        :style="{ height: appStore.mainHeight }">
        <a-card
            type="flex"
            class="app-card app-card--fill">
            <!-- 工具栏 -->
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
                        {{ $t('pages.tagging.add') }}
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            :label="$t('pages.tagging.form.name')"
                            name="name"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.tagging.form.name.placeholder')"
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

            <!-- 数据表格 -->
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
                        <template v-if="'relation' === column.key">
                            <a-tag color="blue">{{ record.relation || 'AND' }}</a-tag>
                        </template>
                        <template v-if="'enabled' === column.key">
                            <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                                {{
                                    record.enabled === 1
                                        ? $t('pages.tagging.form.enabled.active')
                                        : $t('pages.tagging.form.enabled.inactive')
                                }}
                            </a-tag>
                        </template>

                        <template v-if="'created_at' === column.key">
                            {{ formatUtcDateTime(record.created_at) }}
                        </template>

                        <template v-if="'action' === column.key">
                            <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.tagging.edit') }}</template>
                                    <edit-outlined />
                                </a-tooltip>
                            </x-action-button>
                            <x-action-button @click="$refs.editDialogRef.handleCopy(record)">
                                <a-tooltip>
                                    <template #title> {{ $t('pages.policy.copy') }}</template>
                                    <copy-outlined style="color: #52c41a" />
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
            </div>
        </a-card>

        <!-- 配置表单对话框 -->
        <edit-dialog
            ref="editDialogRef"
            @ok="onOk"></edit-dialog>
    </div>
</template>

<script setup>
import { message, Modal } from 'ant-design-vue'
import { ref, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination, useTableAutoScrollY } from '@/hooks'
import { useAppStore } from '@/store'
import EditDialog from './TaggingEditDialog.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'taggingList',
})

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const columns = [
    {
        title: t('pages.tagging.form.name'),
        dataIndex: 'name',
        minWidth: 200,
        ellipsis: {
            showTitle: true,
        },
    },
    { title: t('pages.policy.form.priority') || '冲突优先级', dataIndex: 'priority', width: 130 },
    {
        title: t('pages.tagging.form.relation'),
        dataIndex: 'relation',
        key: 'relation',
        width: 130,
    },
    {
        title: t('pages.tagging.form.enabled'),
        dataIndex: 'enabled',
        key: 'enabled',
        width: 110,
    },
    { title: t('pages.tagging.form.creator'), dataIndex: 'creator', width: 100 },
    { title: t('pages.tagging.form.description'), dataIndex: 'description', width: 250, ellipsis: true },
    {
        title: t('pages.tagging.form.created_at'),
        dataIndex: 'created_at',
        key: 'created_at',
        fixed: 'right',
        width: 180,
        sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 160 },
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

onMounted(() => {
    getPageList()
})

// 监听 query 参数变化，处理 keepAlive 场景下从模型详情页重复跳转
watch(
    () => route.query.policyId,
    (policyId) => {
        if (policyId) {
            nextTick(() => tryOpenEditFromQuery())
        }
    }
)

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.policy
            .getTaggingList({
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
            tryOpenEditFromQuery()
        }
    } catch (error) {
        hideLoading()
    }
}

// 根据路由 query 参数自动打开编辑弹窗（用于从模型详情页跳转）
function tryOpenEditFromQuery() {
    const policyId = route.query.policyId
    if (!policyId) return
    const target = listData.value.find((item) => String(item.id) === String(policyId))
    if (target) {
        editDialogRef.value?.handleEdit(target)
        const query = { ...route.query }
        delete query.policyId
        router.replace({ query })
    }
}

function handleRemove({ id }) {
    Modal.confirm({
        title: t('pages.tagging.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.policy.delTagging(id)
                        if (config('http.code.success') === success) {
                            resolve()
                            message.success(t('component.message.success.delete'))
                            await getPageList()
                        } else {
                            reject()
                        }
                    } catch (error) {
                        // Error message is already shown by interceptor
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
