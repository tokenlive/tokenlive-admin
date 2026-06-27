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
            :label-col="{ style: { width: '100px' } }">
            <div class="loadbalance-form-content">
                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.loadbalance.form.name')"
                            name="name">
                            <a-input v-model:value="formData.name"></a-input>
                        </a-form-item>
                    </a-col>
                </a-row>

                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.loadbalance.form.policyType')"
                            name="type">
                            <a-select
                                :placeholder="$t('pages.loadbalance.form.policyType.placeholder')"
                                v-model:value="formData.type">
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
                    </a-col>
                </a-row>

                <template v-if="formData.type === 'composite'">
                    <a-row :gutter="12">
                        <a-col :span="12">
                            <a-form-item
                                :label="$t('pages.loadbalance.form.costWeight')"
                                name="cost_weight">
                                <a-input-number
                                    v-model:value="formData.cost_weight"
                                    :min="0"
                                    :max="1"
                                    :step="0.1"
                                    :placeholder="$t('pages.loadbalance.form.costWeight.placeholder')"
                                    style="width: 100%" />
                            </a-form-item>
                        </a-col>
                        <a-col :span="12">
                            <a-form-item
                                :label="$t('pages.loadbalance.form.latencyWeight')"
                                name="latency_weight">
                                <a-input-number
                                    v-model:value="formData.latency_weight"
                                    :min="0"
                                    :max="1"
                                    :step="0.1"
                                    :placeholder="$t('pages.loadbalance.form.latencyWeight.placeholder')"
                                    style="width: 100%" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                </template>

                <template v-if="formData.type === 'sticky'">
                    <a-row :gutter="12">
                        <a-col :span="24">
                            <a-form-item
                                :label="$t('pages.loadbalance.form.sessionHeader')"
                                name="session_header">
                                <a-input
                                    v-model:value="formData.session_header"
                                    :placeholder="$t('pages.loadbalance.form.sessionHeader.placeholder')" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                </template>

                <template v-if="formData.type === 'endpoint_affinity'">
                    <a-row :gutter="12">
                        <a-col :span="12">
                            <a-form-item
                                :label="$t('pages.loadbalance.form.sourceType')"
                                name="source_type">
                                <a-select
                                    v-model:value="formData.source_type"
                                    :placeholder="$t('pages.loadbalance.form.sourceType.placeholder')">
                                    <a-select-option value="header">请求头 (header)</a-select-option>
                                    <a-select-option value="query">请求参数 (query)</a-select-option>
                                    <a-select-option value="cookie">Cookie (cookie)</a-select-option>
                                </a-select>
                            </a-form-item>
                        </a-col>
                        <a-col :span="12">
                            <a-form-item
                                :label="$t('pages.loadbalance.form.sourceKey')"
                                name="source_key">
                                <a-input
                                    v-model:value="formData.source_key"
                                    :placeholder="$t('pages.loadbalance.form.sourceKey.placeholder')" />
                            </a-form-item>
                        </a-col>
                    </a-row>
                </template>

                <a-row :gutter="12">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.loadbalance.form.enabled')"
                            name="enabled">
                            <a-switch
                                v-model:checked="enabledSwitch"
                                :checked-children="$t('pages.loadbalance.form.enabled.active')"
                                :un-checked-children="$t('pages.loadbalance.form.enabled.inactive')" />
                        </a-form-item>
                    </a-col>
                </a-row>

                <a-row :gutter="24">
                    <a-col :span="24">
                        <a-form-item
                            :label="$t('pages.loadbalance.form.description')"
                            name="description">
                            <a-textarea v-model:value="formData.description"></a-textarea>
                        </a-form-item>
                    </a-col>
                </a-row>
            </div>
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

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

formRules.value = {
    name: { required: true, message: t('pages.loadbalance.form.name.required') },
    type: { required: true, message: t('pages.loadbalance.form.policyType.required') },
}

const enabledSwitch = computed({
    get: () => formData.value.enabled === 1,
    set: (val) => {
        formData.value.enabled = val ? 1 : 0
    },
})

function handleCreate() {
    formData.value.enabled = 0
    formData.value.cost_weight = 0.5
    formData.value.latency_weight = 0.5
    formData.value.session_header = ''
    formData.value.source_type = 'header'
    formData.value.source_key = ''
    showModal({
        type: 'create',
        title: t('pages.loadbalance.add'),
    })
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.loadbalance.copy'),
    })

    const { data, success } = await apis.policy.getLoadbalance(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }

    data.name = `${data.name} - ${t('pages.policy.copy.suffix')}`
    delete data.id
    delete data.created_at
    delete data.updated_at

    let costWeight = 0.5
    let latencyWeight = 0.5
    let sessionHeader = ''
    let sourceType = 'header'
    let sourceKey = ''
    if (data.params) {
        try {
            const paramsObj = typeof data.params === 'string' ? JSON.parse(data.params) : data.params
            if (paramsObj) {
                if (paramsObj.cost_weight !== undefined) costWeight = paramsObj.cost_weight
                if (paramsObj.latency_weight !== undefined) latencyWeight = paramsObj.latency_weight
                if (paramsObj.session_header !== undefined) sessionHeader = paramsObj.session_header
                if (paramsObj.source_type !== undefined) sourceType = paramsObj.source_type
                if (paramsObj.source_key !== undefined) sourceKey = paramsObj.source_key
            }
        } catch (e) {
            console.error('Failed to parse params json', e)
        }
    }
    data.cost_weight = costWeight
    data.latency_weight = latencyWeight
    data.session_header = sessionHeader
    data.source_type = sourceType
    data.source_key = sourceKey

    formData.value = cloneDeep(data)
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.loadbalance.edit'),
    })

    const { data, success } = await apis.policy.getLoadbalance(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }

    let costWeight = 0.5
    let latencyWeight = 0.5
    let sessionHeader = ''
    let sourceType = 'header'
    let sourceKey = ''
    if (data.params) {
        try {
            const paramsObj = typeof data.params === 'string' ? JSON.parse(data.params) : data.params
            if (paramsObj) {
                if (paramsObj.cost_weight !== undefined) costWeight = paramsObj.cost_weight
                if (paramsObj.latency_weight !== undefined) latencyWeight = paramsObj.latency_weight
                if (paramsObj.session_header !== undefined) sessionHeader = paramsObj.session_header
                if (paramsObj.source_type !== undefined) sourceType = paramsObj.source_type
                if (paramsObj.source_key !== undefined) sourceKey = paramsObj.source_key
            }
        } catch (e) {
            console.error('Failed to parse params json', e)
        }
    }
    data.cost_weight = costWeight
    data.latency_weight = latencyWeight
    data.session_header = sessionHeader
    data.source_type = sourceType
    data.source_key = sourceKey

    formRecord.value = data
    formData.value = cloneDeep(data)
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                const params = {
                    ...values,
                }
                if (values.type === 'composite') {
                    params.params = JSON.stringify({
                        cost_weight: formData.value.cost_weight !== undefined ? formData.value.cost_weight : 0.5,
                        latency_weight:
                            formData.value.latency_weight !== undefined ? formData.value.latency_weight : 0.5,
                    })
                } else if (values.type === 'sticky') {
                    params.params = JSON.stringify({
                        session_header: formData.value.session_header || '',
                    })
                } else if (values.type === 'endpoint_affinity') {
                    params.params = JSON.stringify({
                        source_type: formData.value.source_type || 'header',
                        source_key: formData.value.source_key || '',
                    })
                } else {
                    params.params = null
                }
                delete params.cost_weight
                delete params.latency_weight
                delete params.session_header
                delete params.source_type
                delete params.source_key

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createLoadbalance(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateLoadbalance(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                hideLoading()
                if (config('http.code.success') === result?.success) {
                    hideModal()
                    emit('ok')
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
    hideLoading()
}

defineExpose({
    handleCreate,
    handleEdit,
    handleCopy,
})
</script>

<style lang="less" scoped>
.loadbalance-form-content {
    padding-top: 4px;
}
</style>
