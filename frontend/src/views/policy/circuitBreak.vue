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
                        {{ $t('pages.circuitBreak.add') }}
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            :label="$t('pages.circuitBreak.form.name')"
                            name="name"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.circuitBreak.form.name.placeholder')"
                                v-model:value="searchFormData.name"
                                style="width: 200px"
                                @pressEnter="handleSearch"></a-input>
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
            <a-table
                :columns="columns"
                :data-source="listData"
                :loading="loading"
                :pagination="paginationState"
                :scroll="{ x: 'max-content' }"
                @change="onTableChange">
                <template #bodyCell="{ column, record }">
                    <template v-if="'level' === column.key">
                        <a-tag :color="levelColorMap[record.level] || 'default'">
                            {{ levelMap[record.level] || record.level }}
                        </a-tag>
                    </template>
                    <template v-if="'sliding_window_type' === column.key">
                        {{ slidingWindowTypeMap[record.sliding_window_type] || record.sliding_window_type }}
                    </template>
                    <template v-if="'enabled' === column.key">
                        <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                            {{
                                record.enabled === 1
                                    ? $t('pages.circuitBreak.form.enabled.active')
                                    : $t('pages.circuitBreak.form.enabled.inactive')
                            }}
                        </a-tag>
                    </template>
                    <template v-if="'created_at' === column.key">
                        {{ formatUtcDateTime(record.created_at) }}
                    </template>
                    <template v-if="'action' === column.key">
                        <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                            <a-tooltip>
                                <template #title>
                                    {{ $t('pages.circuitBreak.edit') }}
                                </template>
                                <edit-outlined />
                            </a-tooltip>
                        </x-action-button>
                        <x-action-button @click="$refs.editDialogRef.handleCopy(record)">
                            <a-tooltip>
                                <template #title> {{ $t('pages.policy.copy') }}</template>
                                <copy-outlined style="color: #52c41a" />
                            </a-tooltip>
                        </x-action-button>
                        <x-action-button @click="handleViewBindings(record)">
                            <a-tooltip>
                                <template #title> {{ $t('pages.policy.binding.view') }}</template>
                                <link-outlined style="color: #1890ff" />
                            </a-tooltip>
                        </x-action-button>
                        <x-action-button @click="handleRemove(record)">
                            <a-tooltip>
                                <template #title>
                                    {{ $t('pages.system.delete') }}
                                </template>
                                <delete-outlined style="color: #ff4d4f" />
                            </a-tooltip>
                        </x-action-button>
                    </template>
                </template>
            </a-table>
        </a-card>

        <edit-dialog
            ref="editDialogRef"
            @ok="onOk"></edit-dialog>

        <!-- 策略绑定抽屉 -->
        <policy-binding-drawer
            v-model:visible="bindingDrawerVisible"
            policy-type="circuit_break"
            :policy-id="selectedPolicyId" />
    </div>
</template>

<script setup>
import { message, Modal } from 'ant-design-vue'
import { ref, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './CircuitBreakEditDialog.vue'
import PolicyBindingDrawer from './PolicyBindingDrawer.vue'
import {
    PlusOutlined,
    EditOutlined,
    DeleteOutlined,
    SearchOutlined,
    RedoOutlined,
    LinkOutlined,
    CopyOutlined,
} from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'circuitBreakList',
})
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const columns = [
    {
        title: t('pages.circuitBreak.form.name'),
        dataIndex: 'name',
        minWidth: 200,
        ellipsis: {
            showTitle: true,
        },
    },
    {
        title: t('pages.circuitBreak.form.level'),
        dataIndex: 'level',
        key: 'level',
        width: 80,
    },
    {
        title: t('pages.circuitBreak.form.slidingWindowType'),
        dataIndex: 'sliding_window_type',
        key: 'sliding_window_type',
        width: 100,
    },
    {
        title: t('pages.circuitBreak.form.slowCallMetric'),
        dataIndex: 'slow_call_metric',
        width: 160,
    },
    {
        title: t('pages.circuitBreak.form.enabled'),
        key: 'enabled',
        width: 110,
    },
    { title: t('pages.circuitBreak.form.creator'), dataIndex: 'creator', width: 80 },
    {
        title: t('pages.circuitBreak.form.description'),
        dataIndex: 'description',
        width: 200,
        ellipsis: true,
    },
    {
        title: t('pages.circuitBreak.form.created_at'),
        dataIndex: 'created_at',
        key: 'created_at',
        fixed: 'right',
        width: 180,
        sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    { title: t('button.action'), key: 'action', fixed: 'right', width: 200 },
]

const { listData, loading, showLoading, hideLoading, paginationState, searchFormData, resetPagination } =
    usePagination()
const editDialogRef = ref()
const bindingDrawerVisible = ref(false)
const selectedPolicyId = ref('')
const levelMap = {
    SERVICE: t('pages.circuitBreak.form.level.service'),
    API: t('pages.circuitBreak.form.level.api'),
    INSTANCE: t('pages.circuitBreak.form.level.instance'),
}
const levelColorMap = {
    SERVICE: 'blue',
    API: 'orange',
    INSTANCE: 'green',
}
const slidingWindowTypeMap = {
    time: t('pages.circuitBreak.form.slidingWindowType.time'),
    count: t('pages.circuitBreak.form.slidingWindowType.count'),
}

getPageList()

async function getPageList() {
    try {
        showLoading()
        const { pageSize, current } = paginationState
        const { success, data, total } = await apis.policy
            .getCircuitBreakList({
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

// 监听 query 参数变化，处理 keepAlive 场景下从模型详情页重复跳转
watch(
    () => route.query.policyId,
    (policyId) => {
        if (policyId) {
            nextTick(() => tryOpenEditFromQuery())
        }
    }
)

function handleRemove({ id }) {
    Modal.confirm({
        title: t('pages.circuitBreak.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.policy.delCircuitBreak(id)
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

function handleViewBindings(record) {
    selectedPolicyId.value = record.id
    bindingDrawerVisible.value = true
}
</script>
