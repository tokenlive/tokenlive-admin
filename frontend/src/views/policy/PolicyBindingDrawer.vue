<template>
    <a-drawer
        :title="$t('pages.policy.binding.title')"
        :width="720"
        :open="visible"
        :closable="true"
        @close="handleClose">
        <a-spin :spinning="loading">
            <a-empty
                v-if="!loading && bindingList.length === 0"
                :description="$t('pages.policy.binding.empty')" />
            <div
                v-else
                class="binding-list-wrapper">
                <a-card
                    v-for="item in bindingList"
                    :key="item.id"
                    class="binding-card"
                    size="small">
                    <template #title>
                        <a-space>
                            <a-tag color="blue">{{ $t('pages.policy.binding.dimension') }}</a-tag>
                            <span class="dimension-text">{{ formatDimension(item) }}</span>
                        </a-space>
                    </template>
                    <template #extra>
                        <a-button
                            type="link"
                            danger
                            size="small"
                            @click="handleUnbind(item)">
                            <template #icon><delete-outlined /></template>
                            {{ $t('pages.model.policy.unbind') }}
                        </a-button>
                    </template>
                    <a-descriptions
                        :column="2"
                        size="small"
                        class="binding-desc">
                        <a-descriptions-item :label="$t('pages.policy.binding.tenant')">
                            <a-tag color="cyan">{{ item.tenant_code || $t('pages.policy.binding.all') }}</a-tag>
                        </a-descriptions-item>
                        <a-descriptions-item :label="$t('pages.policy.binding.user')">
                            <a-tag color="purple">{{ item.user_id || $t('pages.policy.binding.all') }}</a-tag>
                        </a-descriptions-item>
                        <a-descriptions-item :label="$t('pages.policy.binding.model')">
                            <a-tag color="green">{{ item.model_code || $t('pages.policy.binding.all') }}</a-tag>
                        </a-descriptions-item>
                        <a-descriptions-item :label="$t('pages.policy.binding.priority')">
                            <a-tag>{{ item.priority }}</a-tag>
                        </a-descriptions-item>
                        <a-descriptions-item
                            :label="$t('pages.policy.binding.enabled')"
                            :span="2">
                            <a-tag :color="item.enabled === 1 ? 'success' : 'default'">
                                {{
                                    item.enabled === 1
                                        ? $t('pages.policy.binding.enabled.active')
                                        : $t('pages.policy.binding.enabled.inactive')
                                }}
                            </a-tag>
                        </a-descriptions-item>
                    </a-descriptions>
                </a-card>
            </div>
        </a-spin>
    </a-drawer>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Modal, message } from 'ant-design-vue'
import { DeleteOutlined } from '@ant-design/icons-vue'
import apis from '@/apis'

const { t } = useI18n()

const props = defineProps({
    visible: {
        type: Boolean,
        default: false,
    },
    policyType: {
        type: String,
        required: true,
    },
    policyId: {
        type: String,
        default: '',
    },
})

const emit = defineEmits(['update:visible'])

const loading = ref(false)
const bindingList = ref([])

watch(
    () => props.visible,
    (newVal) => {
        if (newVal && props.policyId) {
            loadBindings()
        }
    }
)

async function loadBindings() {
    loading.value = true
    try {
        const { data } = await apis.policy.getPolicyBindingList({
            policy_type: props.policyType,
            policy_id: props.policyId,
            pagination: false,
        })
        bindingList.value = data || []
    } catch (error) {
        console.error('Failed to load policy bindings', error)
    } finally {
        loading.value = false
    }
}

function formatDimension(item) {
    const parts = []
    if (item.tenant_code) parts.push(`Tenant: ${item.tenant_code}`)
    if (item.user_id) parts.push(`User: ${item.user_id}`)
    if (item.model_code) parts.push(`Model: ${item.model_code}`)
    return parts.length > 0 ? parts.join(' / ') : t('pages.policy.binding.allDimensions')
}

function handleUnbind(item) {
    Modal.confirm({
        title: t('pages.model.policy.unbindTip'),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: () => {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const { success } = await apis.policy.delPolicyBinding(item.id).catch(() => {
                            throw new Error()
                        })
                        if (success) {
                            resolve()
                            message.success(t('common.success'))
                            await loadBindings()
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

function handleClose() {
    emit('update:visible', false)
}
</script>

<style lang="less" scoped>
.binding-list-wrapper {
    padding: 8px 4px;
}

.binding-card {
    margin-bottom: 16px;
    border-radius: 8px;
    background: var(--color-bg-container);
    border: 1px solid var(--color-border-secondary);
    transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
    box-shadow: var(--shadow-sm);

    &:hover {
        border-color: var(--color-primary);
        box-shadow: var(--shadow-glow);
        transform: translateY(-2px);
    }

    :deep(.ant-card-head) {
        border-bottom: 1px solid var(--color-border-secondary);
        padding: 0 16px;
        min-height: 48px;
        background: var(--color-bg-active);
        border-top-left-radius: 8px;
        border-top-right-radius: 8px;
    }

    :deep(.ant-card-head-title) {
        font-weight: 600;
        font-size: 14px;
        color: var(--color-text-primary);
    }

    :deep(.ant-card-body) {
        padding: 16px;
    }
}

.dimension-text {
    font-size: 14px;
    color: var(--color-text-primary);
    font-weight: 500;
}

.binding-desc {
    :deep(.ant-descriptions-item-label) {
        color: var(--color-text-tertiary);
        font-weight: normal;
        padding-bottom: 8px;
    }

    :deep(.ant-descriptions-item-content) {
        color: var(--color-text-primary);
        padding-bottom: 8px;
        display: flex;
        align-items: center;
    }
}
</style>
