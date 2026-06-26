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
                        {{ $t('pages.loadbalance.add') }}
                    </a-button>
                </a-col>
                <a-col flex="auto"></a-col>
                <a-col flex="none">
                    <a-form
                        :model="searchFormData"
                        layout="inline">
                        <a-form-item
                            :label="$t('pages.loadbalance.form.name')"
                            name="name"
                            style="margin-bottom: 0">
                            <a-input
                                :placeholder="$t('pages.loadbalance.form.name.placeholder')"
                                v-model:value="searchFormData.name"
                                style="width: 200px"
                                @pressEnter="handleSearch"></a-input>
                        </a-form-item>
                        <a-form-item
                            :label="$t('pages.loadbalance.form.policyType')"
                            name="type"
                            style="margin-bottom: 0">
                            <a-select
                                :placeholder="$t('pages.loadbalance.form.policyType.placeholder')"
                                v-model:value="searchFormData.type"
                                allow-clear
                                style="width: 200px">
                                <a-select-option value="round_robin">轮询策略 (round_robin)</a-select-option>
                                <a-select-option value="weighted_rr">加权轮询策略 (weighted_rr)</a-select-option>
                                <a-select-option value="weighted_random"
                                    >权重随机策略 (weighted_random)</a-select-option
                                >
                                <a-select-option value="random">随机策略 (random)</a-select-option>
                                <a-select-option value="least_connections"
                                    >最少连接策略 (least_connections)</a-select-option
                                >
                                <a-select-option value="least_latency">最低延迟策略 (least_latency)</a-select-option>
                                <a-select-option value="cost">最低成本策略 (cost)</a-select-option>
                                <a-select-option value="sticky">会话保持策略 (sticky)</a-select-option>
                                <a-select-option value="composite">综合策略 (composite)</a-select-option>
                                <a-select-option value="endpoint_affinity"
                                    >端点亲和性策略 (endpoint_affinity)</a-select-option
                                >
                            </a-select>
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
                    <template v-if="'type' === column.key">
                        {{ policyTypeMap[record.type] || record.type }}
                    </template>
                    <template v-if="'enabled' === column.key">
                        <a-tag :color="record.enabled === 1 ? 'green' : 'default'">
                            {{
                                record.enabled === 1
                                    ? $t('pages.loadbalance.form.enabled.active')
                                    : $t('pages.loadbalance.form.enabled.inactive')
                            }}
                        </a-tag>
                    </template>

                    <template v-if="'created_at' === column.key">
                        {{ formatUtcDateTime(record.created_at) }}
                    </template>

                    <template v-if="'action' === column.key">
                        <x-action-button @click="$refs.editDialogRef.handleEdit(record)">
                            <a-tooltip>
                                <template #title> {{ $t('pages.loadbalance.edit') }}</template>
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
                                <template #title> {{ $t('pages.system.delete') }}</template>
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
            policy-type="loadbalance"
            :policy-id="selectedPolicyId" />
    </div>
</template>

<script setup>
import { message, Modal } from 'ant-design-vue'
import { ref, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import apis from '@/apis'
import { formatUtcDateTime } from '@/utils/util'
import { config } from '@/config'
import { usePagination } from '@/hooks'
import EditDialog from './LoadbalanceEditDialog.vue'
import PolicyBindingDrawer from './PolicyBindingDrawer.vue'
import { PlusOutlined, EditOutlined, DeleteOutlined, LinkOutlined, CopyOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'loadbalanceList',
})
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const columns = [
    {
        title: t('pages.loadbalance.form.name'),
        dataIndex: 'name',
        minWidth: 200,
        ellipsis: {
            showTitle: true,
        },
    },
    {
        title: t('pages.loadbalance.form.policyType'),
        dataIndex: 'type',
        key: 'type',
        width: 280,
        ellipsis: true,
    },
    { title: t('pages.loadbalance.form.enabled'), key: 'enabled', width: 110 },
    { title: t('pages.loadbalance.form.creator'), dataIndex: 'creator', width: 100 },
    { title: t('pages.loadbalance.form.description'), dataIndex: 'description', width: 250, ellipsis: true },
    {
        title: t('pages.loadbalance.form.created_at'),
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
const policyTypeMap = {
    round_robin: '轮询策略 (round_robin)',
    weighted_rr: '加权轮询策略 (weighted_rr)',
    weighted_random: '权重随机策略 (weighted_random)',
    random: '随机策略 (random)',
    least_connections: '最少连接策略 (least_connections)',
    least_latency: '最低延迟策略 (least_latency)',
    cost: '最低成本策略 (cost)',
    sticky: '会话保持策略 (sticky)',
    composite: '综合策略 (composite)',
    endpoint_affinity: '端点亲和性策略 (endpoint_affinity)',
}

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
            .getLoadbalanceList({
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
        title: t('pages.loadbalance.delTip'),
        content: t('button.confirm'),
        okText: t('button.confirm'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.policy.delLoadbalance(id)
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

// Table 分页改变
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
