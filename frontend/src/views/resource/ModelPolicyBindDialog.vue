<template>
    <a-modal
        :open="modal.open"
        :title="modal.title"
        :width="600"
        :confirm-loading="modal.confirmLoading"
        :after-close="onAfterClose"
        :cancel-text="cancelText"
        :ok-text="okText"
        @ok="handleOk"
        @cancel="handleCancel">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '110px' } }">
            <a-card
                class="mb-4"
                :bordered="false"
                style="padding: 0">
                <!-- 策略类型 -->
                <a-form-item
                    :label="$t('pages.model.policy.form.policy_type')"
                    name="policy_type">
                    <a-select
                        v-model:value="formData.policy_type"
                        :placeholder="$t('pages.model.policy.form.policy_type.placeholder')"
                        :disabled="modal.type === 'edit'"
                        @change="handlePolicyTypeChange">
                        <a-select-option value="tagging">
                            {{ $t('pages.dashboard.policies.tagging') }} (tagging)
                        </a-select-option>
                        <a-select-option value="limit">
                            {{ $t('pages.dashboard.policies.limit') }} (limit)
                        </a-select-option>
                        <a-select-option value="invocation">
                            {{ $t('pages.dashboard.policies.invocation') }} (invocation)
                        </a-select-option>
                        <a-select-option value="route">
                            {{ $t('pages.dashboard.policies.route') }} (route)
                        </a-select-option>
                        <a-select-option value="loadbalance">
                            {{ $t('pages.dashboard.policies.loadbalance') }} (loadbalance)
                        </a-select-option>
                        <a-select-option value="circuit_break">
                            {{ $t('pages.dashboard.policies.circuitBreak') }} (circuit_break)
                        </a-select-option>
                    </a-select>
                </a-form-item>

                <!-- 治理策略选择 -->
                <a-form-item
                    :label="$t('pages.model.policy.form.policy_id')"
                    name="policy_id">
                    <div style="display: flex; gap: 8px; align-items: center">
                        <a-select
                            v-model:value="formData.policy_id"
                            :placeholder="$t('pages.model.policy.form.policy_id.placeholder')"
                            :loading="policiesLoading"
                            show-search
                            option-filter-prop="label"
                            :disabled="!formData.policy_type"
                            style="width: 320px">
                            <a-select-option
                                v-for="item in policyOptions"
                                :key="item.id"
                                :value="item.id"
                                :label="item.name"
                                :title="item.name">
                                {{ item.name }}
                            </a-select-option>
                        </a-select>
                        <a-button
                            type="primary"
                            ghost
                            :disabled="!formData.policy_type"
                            @click="handleGoToCreatePolicy">
                            <template #icon>
                                <plus-outlined></plus-outlined>
                            </template>
                            新建
                        </a-button>
                    </div>
                </a-form-item>

                <!-- 维度：租户编码 -->
                <a-form-item
                    :label="$t('pages.model.policy.form.tenant_code')"
                    name="tenant_code">
                    <a-input
                        v-model:value="formData.tenant_code"
                        :placeholder="$t('pages.model.policy.form.tenant_code.placeholder')">
                    </a-input>
                </a-form-item>

                <!-- 维度：用户 ID -->
                <a-form-item
                    :label="$t('pages.model.policy.form.user_id')"
                    name="user_id">
                    <a-input
                        v-model:value="formData.user_id"
                        :placeholder="$t('pages.model.policy.form.user_id.placeholder')">
                    </a-input>
                </a-form-item>

                <!-- 优先级 -->
                <a-form-item
                    :label="$t('pages.model.policy.form.priority')"
                    name="priority">
                    <a-input-number
                        v-model:value="formData.priority"
                        :min="0"
                        style="width: 100%"
                        :placeholder="$t('pages.model.policy.form.priority.placeholder')">
                    </a-input-number>
                </a-form-item>

                <!-- 启用状态 -->
                <a-form-item
                    :label="$t('pages.model.policy.form.enabled')"
                    name="enabled">
                    <a-switch
                        v-model:checked="formData.enabled"
                        :checked-value="1"
                        :un-checked-value="0">
                    </a-switch>
                </a-form-item>

                <!-- 备注 -->
                <a-form-item
                    :label="$t('pages.provider.form.description')"
                    name="description">
                    <a-textarea
                        v-model:value="formData.description"
                        :placeholder="$t('pages.provider.form.description.placeholder')">
                    </a-textarea>
                </a-form-item>

                <!-- 排他性策略警告提示 -->
                <div
                    v-if="isExclusivePolicy"
                    style="margin-top: 16px">
                    <a-alert
                        type="warning"
                        show-icon
                        :message="$t('pages.model.policy.form.exclusive.warning')" />
                </div>
            </a-card>
        </a-form>
    </a-modal>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { message } from 'ant-design-vue'
import { ref, computed } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { PlusOutlined } from '@ant-design/icons-vue'

const props = defineProps({
    modelCode: {
        type: String,
        required: true,
    },
})

const emit = defineEmits(['ok'])
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const router = useRouter()

const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

function handleGoToCreatePolicy() {
    const policyType = formData.value.policy_type
    if (!policyType) {
        return
    }

    const routeMap = {
        tagging: 'taggingList',
        limit: 'limitList',
        invocation: 'invocationList',
        route: 'routeList',
        loadbalance: 'loadbalanceList',
        circuit_break: 'circuitBreakList',
    }

    const routeName = routeMap[policyType]
    if (routeName) {
        hideModal()
        router.push({ name: routeName })
    }
}

const policyOptions = ref([])
const policiesLoading = ref(false)

formRules.value = {
    policy_type: { required: true, message: t('pages.model.policy.form.policy_type.required'), trigger: 'change' },
    policy_id: { required: true, message: t('pages.model.policy.form.policy_id.required'), trigger: 'change' },
}

const isExclusivePolicy = computed(() => {
    return formData.value.policy_type === 'loadbalance' || formData.value.policy_type === 'invocation'
})

function handleCreate() {
    showModal({
        type: 'create',
        title: t('pages.model.policy.bind'),
    })
    formData.value.model_code = props.modelCode
    formData.value.enabled = 1
    formData.value.priority = 0
    policyOptions.value = []
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.endpoint.edit'),
    })

    try {
        const { data, success } = await apis.policy.getPolicyBinding(record.id).catch()
        if (!success) {
            message.error(t('component.message.error.save'))
            hideModal()
            return
        }
        formRecord.value = data
        formData.value = cloneDeep(data)
        if (data.policy_type) {
            await loadPolicies(data.policy_type)
        }
    } catch (e) {
        hideModal()
    }
}

async function handlePolicyTypeChange(policyType) {
    formData.value.policy_id = undefined
    policyOptions.value = []
    if (policyType) {
        await loadPolicies(policyType)
    }
}

async function loadPolicies(policyType) {
    try {
        policiesLoading.value = true
        let res = null
        const params = { pageSize: 1000, current: 1 }

        switch (policyType) {
            case 'tagging':
                res = await apis.policy.getTaggingList(params)
                break
            case 'limit':
                res = await apis.policy.getLimitList(params)
                break
            case 'invocation':
                res = await apis.policy.getInvocationList(params)
                break
            case 'route':
                res = await apis.policy.getRouteList(params)
                break
            case 'loadbalance':
                res = await apis.policy.getLoadbalanceList(params)
                break
            case 'circuit_break':
                res = await apis.policy.getCircuitBreakList(params)
                break
        }

        if (res && config('http.code.success') === res.success) {
            policyOptions.value = res.data || []
        }
    } catch (e) {
        // ignore
    } finally {
        policiesLoading.value = false
    }
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                const params = {
                    ...values,
                    model_code: props.modelCode,
                    tenant_code: formData.value.tenant_code || '',
                    user_id: formData.value.user_id || '',
                    description: formData.value.description,
                    priority: formData.value.priority || 0,
                    enabled: formData.value.enabled,
                }

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createPolicyBinding(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updatePolicyBinding(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')
                } else if (result?.message) {
                    message.error(result.message)
                }
            } catch (error) {
                hideLoading()
            }
        })
        .catch(() => {
            hideLoading()
        })
}

function handleCancel() {
    hideModal()
}

function onAfterClose() {
    resetForm()
    policyOptions.value = []
    hideLoading()
}

defineExpose({
    handleCreate,
    handleEdit,
})
</script>

<style lang="less" scoped>
.mb-4 {
    margin-bottom: 16px;
}
</style>
