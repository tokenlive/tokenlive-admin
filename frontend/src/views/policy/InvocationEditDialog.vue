<template>
    <a-drawer
        :open="modal.open"
        :title="modal.title"
        :width="720"
        placement="right"
        :closable="true"
        @close="handleCancel">
        <a-form
            ref="formRef"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '140px' } }"
            :wrapper-col="{ flex: 1 }"
            :class="{ 'dark-form': appStore.config.theme === 'dark' }">
            <!-- 规则名称 -->
            <a-form-item
                :label="$t('pages.invocation.form.name')"
                name="name">
                <a-input
                    :placeholder="$t('pages.invocation.form.name.placeholder')"
                    v-model:value="formData.name"
                    :maxlength="60" />
            </a-form-item>

            <!-- 调用类别 -->
            <a-form-item
                :label="$t('pages.invocation.form.type')"
                name="type">
                <a-radio-group
                    v-model:value="formData.type"
                    @change="onTypeChange">
                    <a-radio value="failfast">{{ $t('pages.invocation.form.type.failfast') }}</a-radio>
                    <a-radio value="failover">{{ $t('pages.invocation.form.type.failover') }}</a-radio>
                </a-radio-group>
            </a-form-item>

            <!-- 错误判断条件 (仅 failover) -->
            <template v-if="formData.type === 'failover'">
                <a-form-item
                    :label="$t('pages.invocation.form.errorCondition')"
                    name="errorCodes"
                    class="error-condition-form-item">
                    <template #label>
                        <span>
                            {{ $t('pages.invocation.form.errorCondition') }}
                            <a-tooltip :title="$t('pages.invocation.form.errorCondition.tooltip')">
                                <question-circle-outlined style="margin-left: 4px; color: #999" />
                            </a-tooltip>
                        </span>
                    </template>
                    <div class="error-condition-section">
                        <div
                            style="
                                background: #e6f7ff;
                                border: 1px solid #91d5ff;
                                border-radius: 4px;
                                padding: 8px 12px;
                                margin-bottom: 16px;
                                font-size: 13px;
                                color: #1890ff;
                            ">
                            {{ $t('pages.invocation.form.errorCondition.tip') }}
                        </div>

                        <!-- 错误码 -->
                        <div class="condition-row">
                            <span class="condition-label">
                                {{ $t('pages.invocation.form.errorCodes') }}
                            </span>
                            <a-select
                                v-model:value="formData.errorCodes"
                                mode="tags"
                                :placeholder="$t('pages.invocation.form.errorCodes.placeholder')"
                                style="flex: 1" />
                        </div>

                        <!-- 错误消息 -->
                        <div
                            class="condition-row"
                            style="margin-top: 16px">
                            <span class="condition-label">
                                {{ $t('pages.invocation.form.errorMessages') }}
                            </span>
                            <a-select
                                v-model:value="formData.errorMessages"
                                mode="tags"
                                :placeholder="$t('pages.invocation.form.errorMessages.placeholder')"
                                style="flex: 1" />
                        </div>

                        <!-- 错误码解析策略 -->
                        <div
                            style="
                                margin-top: 16px;
                                border-top: 1px dashed var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
                                padding-top: 16px;
                            ">
                            <div style="font-weight: 500; margin-bottom: 8px">
                                {{ $t('pages.invocation.form.codePolicy.title') }}
                            </div>
                            <div style="display: flex; flex-direction: column; gap: 8px">
                                <div class="condition-row">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.codePolicy.parserType')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.codePolicy.parser"
                                        :placeholder="$t('pages.invocation.form.codePolicy.parserType.placeholder')"
                                        style="flex: 1">
                                        <a-select-option value="">{{
                                            $t('pages.invocation.form.codePolicy.parserType.none')
                                        }}</a-select-option>
                                        <a-select-option value="JsonPath">JsonPath</a-select-option>
                                        <a-select-option value="Regexp">Regexp</a-select-option>
                                    </a-select>
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.codePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.codePolicy.expression')
                                    }}</span>
                                    <a-input
                                        v-model:value="formData.codePolicy.expression"
                                        :placeholder="$t('pages.invocation.form.codePolicy.expression.placeholder')"
                                        style="flex: 1" />
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.codePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.codePolicy.statuses')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.codePolicy.statuses"
                                        mode="tags"
                                        :placeholder="$t('pages.invocation.form.codePolicy.statuses.placeholder')"
                                        style="flex: 1" />
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.codePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.codePolicy.contentTypes')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.codePolicy.contentTypes"
                                        mode="tags"
                                        :placeholder="$t('pages.invocation.form.codePolicy.contentTypes.placeholder')"
                                        style="flex: 1">
                                        <a-select-option value="application/json">application/json</a-select-option>
                                        <a-select-option value="application/x-www-form-urlencoded"
                                            >application/x-www-form-urlencoded</a-select-option
                                        >
                                        <a-select-option value="text/event-stream">text/event-stream</a-select-option>
                                        <a-select-option value="text/plain">text/plain</a-select-option>
                                    </a-select>
                                </div>
                            </div>
                        </div>

                        <!-- 错误消息解析策略 -->
                        <div
                            style="
                                margin-top: 16px;
                                border-top: 1px dashed var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
                                padding-top: 16px;
                            ">
                            <div style="font-weight: 500; margin-bottom: 8px">
                                {{ $t('pages.invocation.form.messagePolicy.title') }}
                            </div>
                            <div style="display: flex; flex-direction: column; gap: 8px">
                                <div class="condition-row">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.messagePolicy.parserType')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.messagePolicy.parser"
                                        :placeholder="$t('pages.invocation.form.messagePolicy.parserType.placeholder')"
                                        style="flex: 1">
                                        <a-select-option value="">{{
                                            $t('pages.invocation.form.messagePolicy.parserType.none')
                                        }}</a-select-option>
                                        <a-select-option value="JsonPath">JsonPath</a-select-option>
                                        <a-select-option value="Regexp">Regexp</a-select-option>
                                    </a-select>
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.messagePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.messagePolicy.expression')
                                    }}</span>
                                    <a-input
                                        v-model:value="formData.messagePolicy.expression"
                                        :placeholder="$t('pages.invocation.form.messagePolicy.expression.placeholder')"
                                        style="flex: 1" />
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.messagePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.messagePolicy.statuses')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.messagePolicy.statuses"
                                        mode="tags"
                                        :placeholder="$t('pages.invocation.form.messagePolicy.statuses.placeholder')"
                                        style="flex: 1" />
                                </div>
                                <div
                                    class="condition-row"
                                    v-if="formData.messagePolicy.parser">
                                    <span class="condition-label">{{
                                        $t('pages.invocation.form.messagePolicy.contentTypes')
                                    }}</span>
                                    <a-select
                                        v-model:value="formData.messagePolicy.contentTypes"
                                        mode="tags"
                                        :placeholder="
                                            $t('pages.invocation.form.messagePolicy.contentTypes.placeholder')
                                        "
                                        style="flex: 1">
                                        <a-select-option value="application/json">application/json</a-select-option>
                                        <a-select-option value="application/x-www-form-urlencoded"
                                            >application/x-www-form-urlencoded</a-select-option
                                        >
                                        <a-select-option value="text/event-stream">text/event-stream</a-select-option>
                                        <a-select-option value="text/plain">text/plain</a-select-option>
                                    </a-select>
                                </div>
                            </div>
                        </div>
                    </div>
                </a-form-item>

                <!-- 重试策略 -->
                <a-form-item
                    :label="$t('pages.invocation.form.retryPolicy')"
                    class="retry-policy-form-item">
                    <div class="retry-policy-section">
                        <!-- 重试次数 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">{{
                                $t('pages.invocation.form.retryPolicy.retry')
                            }}</span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.retry"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.retryPolicy.retry.placeholder')" />
                                <span class="scheme-unit">{{
                                    $t('pages.invocation.form.retryPolicy.retry.unit')
                                }}</span>
                            </div>
                        </div>
                        <!-- 退避类型 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">{{ $t('pages.invocation.form.backoffType') }}</span>
                            <div class="scheme-input">
                                <a-select
                                    v-model:value="formData.backoffType"
                                    :placeholder="$t('pages.invocation.form.backoffType.placeholder')"
                                    style="width: 100%">
                                    <a-select-option value="fixed">{{
                                        $t('pages.invocation.form.backoffType.fixed')
                                    }}</a-select-option>
                                    <a-select-option value="exponential">{{
                                        $t('pages.invocation.form.backoffType.exponential')
                                    }}</a-select-option>
                                </a-select>
                            </div>
                        </div>
                        <!-- 退避间隔 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">{{ $t('pages.invocation.form.baseMs') }}</span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.baseMs"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.baseMs.placeholder')" />
                                <span class="scheme-unit">ms</span>
                            </div>
                        </div>
                        <!-- 建立连接超时 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">
                                {{ $t('pages.invocation.form.connectTimeout') }}
                                <a-tooltip :title="$t('pages.invocation.form.connectTimeout.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.connectTimeout"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.connectTimeout.placeholder')" />
                                <span class="scheme-unit">ms</span>
                            </div>
                        </div>
                        <!-- 首字超时 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">
                                {{ $t('pages.invocation.form.ttftTimeout') }}
                                <a-tooltip :title="$t('pages.invocation.form.ttftTimeout.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.ttftTimeout"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.ttftTimeout.placeholder')" />
                                <span class="scheme-unit">ms</span>
                            </div>
                        </div>

                        <!-- 读空闲超时 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">
                                {{ $t('pages.invocation.form.idleTimeout') }}
                                <a-tooltip :title="$t('pages.invocation.form.idleTimeout.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.idleTimeout"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.idleTimeout.placeholder')" />
                                <span class="scheme-unit">ms</span>
                            </div>
                        </div>

                        <!-- 请求总超时 -->
                        <div class="scheme-row">
                            <span class="scheme-label required">
                                {{ $t('pages.invocation.form.totalTimeout') }}
                                <a-tooltip :title="$t('pages.invocation.form.totalTimeout.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </span>
                            <div class="scheme-input">
                                <a-input-number
                                    v-model:value="formData.totalTimeout"
                                    :min="0"
                                    style="width: 100%"
                                    :placeholder="$t('pages.invocation.form.totalTimeout.placeholder')" />
                                <span class="scheme-unit">ms</span>
                            </div>
                        </div>
                    </div>
                </a-form-item>
            </template>

            <!-- 模型降级配置 -->
            <a-form-item
                :label="$t('pages.invocation.form.fallbackPolicy')"
                name="fallbackTargets">
                <template #label>
                    <span>
                        {{ $t('pages.invocation.form.fallbackPolicy') }}
                        <a-tooltip :title="$t('pages.invocation.form.fallbackTargets.tooltip')">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="retry-policy-section">
                    <div class="scheme-row">
                        <span class="scheme-label">{{ $t('pages.invocation.form.fallbackTargets') }}</span>
                        <div
                            class="scheme-input"
                            style="flex-direction: column; align-items: flex-start; gap: 12px">
                            <a-select
                                v-model:value="formData.fallbackTargets"
                                mode="tags"
                                :options="modelOptions"
                                :max-tag-count="'responsive'"
                                :placeholder="$t('pages.invocation.form.fallbackTargets.placeholder')"
                                style="width: 100%" />

                            <!-- 降级顺序列表 -->
                            <div
                                v-if="formData.fallbackTargets && formData.fallbackTargets.length > 0"
                                class="fallback-list">
                                <div
                                    v-for="(item, index) in formData.fallbackTargets"
                                    :key="item"
                                    class="fallback-item">
                                    <div class="fallback-item-left">
                                        <span class="fallback-index">{{ String(index + 1).padStart(2, '0') }}</span>
                                        <span class="fallback-name">{{ getModelLabel(item) }}</span>
                                    </div>
                                    <div class="fallback-actions">
                                        <a-tooltip :title="$t('pages.policy.sort.up') || '上移'">
                                            <a-button
                                                type="link"
                                                size="small"
                                                :disabled="index === 0"
                                                @click="moveUp(index)">
                                                <arrow-up-outlined />
                                            </a-button>
                                        </a-tooltip>
                                        <a-tooltip :title="$t('pages.policy.sort.down') || '下移'">
                                            <a-button
                                                type="link"
                                                size="small"
                                                :disabled="index === formData.fallbackTargets.length - 1"
                                                @click="moveDown(index)">
                                                <arrow-down-outlined />
                                            </a-button>
                                        </a-tooltip>
                                        <a-tooltip :title="$t('common.delete') || '删除'">
                                            <a-button
                                                type="link"
                                                size="small"
                                                danger
                                                @click="removeFallback(index)">
                                                <delete-outlined />
                                            </a-button>
                                        </a-tooltip>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </a-form-item>

            <!-- 生效状态 -->
            <a-form-item
                :label="$t('pages.invocation.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="enabledSwitch"
                    :checked-children="$t('pages.invocation.form.enabled.active')"
                    :un-checked-children="$t('pages.invocation.form.enabled.inactive')" />
            </a-form-item>

            <!-- 描述 -->
            <a-form-item
                :label="$t('pages.invocation.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :placeholder="$t('pages.invocation.form.description.placeholder')"
                    :maxlength="255"
                    show-count
                    :rows="3" />
            </a-form-item>
        </a-form>

        <template #footer>
            <div style="text-align: right">
                <a-space>
                    <a-button @click="handleCancel">{{ cancelText }}</a-button>
                    <a-button
                        type="primary"
                        :loading="modal.confirmLoading"
                        @click="handleOk">
                        {{ okText }}
                    </a-button>
                </a-space>
            </div>
        </template>
    </a-drawer>
</template>

<script setup>
import { cloneDeep } from 'lodash-es'
import { message } from 'ant-design-vue'
import { ref, computed, watch } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { QuestionCircleOutlined, ArrowUpOutlined, ArrowDownOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'

const appStore = useAppStore()

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

const modelOptions = ref([])

async function loadModelOptions() {
    try {
        const { success, data } = await apis.model.getModelList({ pageSize: 100, current: 1 }).catch(() => {
            throw new Error()
        })
        if (config('http.code.success') === success) {
            modelOptions.value = (data || []).map((item) => ({
                label: item.model_name ? `${item.model_name} (${item.model_code})` : item.model_code,
                value: item.model_code,
            }))
        }
    } catch (error) {
        // ignore
    }
}

function getModelLabel(value) {
    const opt = modelOptions.value.find((item) => item.value === value)
    return opt ? opt.label : value
}

function moveUp(index) {
    if (index === 0) return
    const targets = [...formData.value.fallbackTargets]
    const temp = targets[index]
    targets[index] = targets[index - 1]
    targets[index - 1] = temp
    formData.value.fallbackTargets = targets
}

function moveDown(index) {
    if (index === formData.value.fallbackTargets.length - 1) return
    const targets = [...formData.value.fallbackTargets]
    const temp = targets[index]
    targets[index] = targets[index + 1]
    targets[index + 1] = temp
    formData.value.fallbackTargets = targets
}

function removeFallback(index) {
    formData.value.fallbackTargets.splice(index, 1)
}

formRules.value = {
    name: { required: true, message: t('pages.invocation.form.name.required') },
    type: { required: true, message: t('pages.invocation.form.type.required') },
    errorCodes: {
        validator: () => {
            if (formData.value.type === 'failover') {
                const hasErrorCodes = formData.value.errorCodes && formData.value.errorCodes.length > 0
                const hasErrorMessages = formData.value.errorMessages && formData.value.errorMessages.length > 0

                if (!hasErrorCodes && !hasErrorMessages) {
                    return Promise.reject(t('pages.invocation.form.errorCondition.required'))
                }
            }
            return Promise.resolve()
        },
    },
}

const enabledSwitch = computed({
    get: () => formData.value.enabled === 1,
    set: (val) => {
        formData.value.enabled = val ? 1 : 0
    },
})

watch(
    () => formData.value.errorCodes,
    () => {
        formRef.value?.validateFields(['errorCodes', 'errorMessages']).catch(() => {})
    },
    { deep: true }
)

watch(
    () => formData.value.errorMessages,
    () => {
        formRef.value?.validateFields(['errorCodes', 'errorMessages']).catch(() => {})
    },
    { deep: true }
)

function onTypeChange() {
    formData.value.errorCodes = []
    formData.value.retry = undefined
    formData.value.backoffType = undefined
    formData.value.baseMs = undefined
    formData.value.connectTimeout = undefined
    formData.value.ttftTimeout = undefined
    formData.value.totalTimeout = undefined
    formData.value.idleTimeout = undefined
}

function handleCreate() {
    loadModelOptions()
    formData.value.fallbackTargets = []
    formData.value.enabled = 0
    formData.value.type = 'failfast'
    formData.value.errorCodes = []
    formData.value.errorMessages = []
    formData.value.codePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    formData.value.messagePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    formData.value.retry = 3
    formData.value.backoffType = 'fixed'
    formData.value.baseMs = 200
    formData.value.connectTimeout = 2000
    formData.value.ttftTimeout = 5000

    formData.value.totalTimeout = 60000
    formData.value.idleTimeout = 0
    showModal({
        type: 'create',
        title: t('pages.invocation.add'),
    })
}

async function handleCopy(record = {}) {
    loadModelOptions()
    showModal({
        type: 'create',
        title: t('pages.invocation.copy'),
    })

    const { data, success } = await apis.policy.getInvocation(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    // 复制模式：不设置 formRecord，让它作为新建处理
    const cloned = cloneDeep(data)
    // 修改名称以区分复制
    cloned.name = `${cloned.name} - ${t('pages.policy.copy.suffix')}`
    delete cloned.id
    delete cloned.created_at
    delete cloned.updated_at
    populateFormData(cloned)
}

async function handleEdit(record = {}) {
    loadModelOptions()
    showModal({
        type: 'edit',
        title: t('pages.invocation.edit'),
    })

    const { data, success } = await apis.policy.getInvocation(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    formRecord.value = data
    const cloned = cloneDeep(data)
    populateFormData(cloned)
}

function populateFormData(cloned) {
    if (typeof cloned.retry_policy === 'string') {
        try {
            cloned.retry_policy = JSON.parse(cloned.retry_policy)
        } catch {
            cloned.retry_policy = null
        }
    }
    if (cloned.retry_policy) {
        cloned.retry = cloned.retry_policy.retry
        cloned.backoffType = cloned.retry_policy.backoff_type || cloned.retry_policy.backoffType || 'fixed'
        cloned.baseMs =
            cloned.retry_policy.base_ms !== undefined
                ? cloned.retry_policy.base_ms
                : cloned.retry_policy.baseMs !== undefined
                  ? cloned.retry_policy.baseMs
                  : cloned.retry_policy.interval || 200
        cloned.connectTimeout =
            cloned.retry_policy.connect_timeout !== undefined
                ? cloned.retry_policy.connect_timeout
                : cloned.retry_policy.connectTimeout
        cloned.ttftTimeout =
            cloned.retry_policy.ttft_timeout !== undefined
                ? cloned.retry_policy.ttft_timeout
                : cloned.retry_policy.ttftTimeout

        cloned.totalTimeout =
            cloned.retry_policy.total_timeout !== undefined
                ? cloned.retry_policy.total_timeout
                : cloned.retry_policy.totalTimeout
        cloned.idleTimeout =
            cloned.retry_policy.idle_timeout !== undefined
                ? cloned.retry_policy.idle_timeout
                : cloned.retry_policy.idleTimeout !== undefined
                  ? cloned.retry_policy.idleTimeout
                  : 0
        cloned.errorCodes = (cloned.retry_policy.error_codes || cloned.retry_policy.errorCodes || []).map(String)
        cloned.errorMessages = (cloned.retry_policy.error_messages || cloned.retry_policy.errorMessages || []).map(
            String
        )

        const rawCodePolicy = cloned.retry_policy.code_policy || cloned.retry_policy.codePolicy
        cloned.codePolicy = rawCodePolicy || {
            parser: '',
            expression: '',
            statuses: [],
            contentTypes: ['application/json'],
        }
        if (rawCodePolicy) {
            cloned.codePolicy.contentTypes = rawCodePolicy.content_types ||
                rawCodePolicy.contentTypes || ['application/json']
        }
        if (!cloned.codePolicy.statuses) cloned.codePolicy.statuses = []

        const rawMessagePolicy = cloned.retry_policy.message_policy || cloned.retry_policy.messagePolicy
        cloned.messagePolicy = rawMessagePolicy || {
            parser: '',
            expression: '',
            statuses: [],
            contentTypes: ['application/json'],
        }
        if (rawMessagePolicy) {
            cloned.messagePolicy.contentTypes = rawMessagePolicy.content_types ||
                rawMessagePolicy.contentTypes || ['application/json']
        }
        if (!cloned.messagePolicy.statuses) cloned.messagePolicy.statuses = []
    } else {
        cloned.retry = 3
        cloned.backoffType = 'fixed'
        cloned.baseMs = 200
        cloned.connectTimeout = 2000
        cloned.ttftTimeout = 5000

        cloned.totalTimeout = 60000
        cloned.idleTimeout = 0
        cloned.errorCodes = []
        cloned.errorMessages = []
        cloned.codePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
        cloned.messagePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    }

    if (typeof cloned.fallback_policy === 'string') {
        try {
            cloned.fallback_policy = JSON.parse(cloned.fallback_policy)
        } catch {
            cloned.fallback_policy = null
        }
    }
    if (cloned.fallback_policy) {
        cloned.fallbackTargets = cloned.fallback_policy.targets || []
    } else {
        cloned.fallbackTargets = []
    }

    formData.value = cloned
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()
                let retryPolicy = undefined
                if (formData.value.type === 'failover') {
                    const codes = (formData.value.errorCodes || []).map(String)
                    const messages = (formData.value.errorMessages || []).map(String)

                    let codePolicy = undefined
                    if (formData.value.codePolicy && formData.value.codePolicy.parser) {
                        codePolicy = {
                            parser: formData.value.codePolicy.parser,
                            expression: formData.value.codePolicy.expression,
                            statuses: formData.value.codePolicy.statuses || [],
                            content_types: formData.value.codePolicy.contentTypes || [],
                        }
                    }

                    let messagePolicy = undefined
                    if (formData.value.messagePolicy && formData.value.messagePolicy.parser) {
                        messagePolicy = {
                            parser: formData.value.messagePolicy.parser,
                            expression: formData.value.messagePolicy.expression,
                            statuses: formData.value.messagePolicy.statuses || [],
                            content_types: formData.value.messagePolicy.contentTypes || [],
                        }
                    }

                    retryPolicy = {
                        retry: formData.value.retry || 0,
                        backoff_type: formData.value.backoffType || 'fixed',
                        base_ms: formData.value.baseMs || 0,
                        error_codes: codes,
                        error_messages: messages,
                        code_policy: codePolicy,
                        message_policy: messagePolicy,
                        connect_timeout: formData.value.connectTimeout || 0,
                        ttft_timeout: formData.value.ttftTimeout || 0,
                        total_timeout: formData.value.totalTimeout || 0,
                        idle_timeout: formData.value.idleTimeout || 0,
                    }
                }
                let fallbackPolicy = undefined
                if (formData.value.fallbackTargets && formData.value.fallbackTargets.length > 0) {
                    fallbackPolicy = {
                        targets: formData.value.fallbackTargets.map(String),
                    }
                }
                const params = {
                    ...values,
                    retry_policy: retryPolicy,
                    fallback_policy: fallbackPolicy,
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createInvocation(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateInvocation(formData.value.id, params).catch(() => {
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
    onAfterClose()
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
.error-condition-form-item,
.retry-policy-form-item {
    :deep(.ant-form-item-control-input-content) {
        overflow: visible;
    }
}

.error-condition-section {
    border: 1px solid var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
    border-radius: 6px;
    padding: 16px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.condition-row {
    display: flex;
    align-items: center;
    gap: 8px;
}

.condition-label {
    font-size: 13px;
    white-space: nowrap;
    min-width: 80px;
    display: flex;
    align-items: center;
}

.retry-policy-section {
    border: 1px solid var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
    border-radius: 6px;
    padding: 16px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.scheme-row {
    display: flex;
    align-items: center;
    margin-bottom: 16px;

    &:last-child {
        margin-bottom: 0;
    }
}

.scheme-label {
    font-size: 13px;
    white-space: nowrap;
    margin-right: 12px;
    min-width: 110px;
    display: flex;
    align-items: center;

    &.required::before {
        content: '* ';
        color: #ff4d4f;
    }
}

.scheme-input {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
}

.scheme-unit {
    opacity: 0.6;
    font-size: 13px;
    white-space: nowrap;
}

.fallback-list {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
}

.fallback-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 6px 12px;
    background: rgba(0, 0, 0, 0.02);
    border: 1px solid var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
    border-radius: 6px;
    transition: all 0.3s ease;

    &:hover {
        background: rgba(24, 144, 255, 0.03);
        border-color: rgba(24, 144, 255, 0.3);
    }

    .fallback-item-left {
        display: flex;
        align-items: center;
        gap: 12px;
        overflow: hidden;
    }

    .fallback-index {
        font-family: monospace;
        font-weight: bold;
        color: #1890ff;
        background: rgba(24, 144, 255, 0.1);
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 12px;
    }

    .fallback-name {
        font-size: 13px;
        color: rgba(0, 0, 0, 0.85);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .fallback-actions {
        display: flex;
        align-items: center;
        gap: 4px;

        :deep(.ant-btn-link) {
            padding: 0 4px;
            height: auto;
        }
    }
}

.dark-form {
    .error-condition-section,
    .retry-policy-section {
        color: rgba(255, 255, 255, 0.85);
        border-color: rgba(255, 255, 255, 0.15);
    }
    .condition-label,
    .scheme-label {
        color: rgba(255, 255, 255, 0.85);
    }
    .scheme-unit {
        color: rgba(255, 255, 255, 0.45);
    }
    .fallback-item {
        background: rgba(255, 255, 255, 0.02);

        &:hover {
            background: rgba(24, 144, 255, 0.05);
            border-color: rgba(24, 144, 255, 0.4);
        }

        .fallback-name {
            color: rgba(255, 255, 255, 0.85);
        }
    }
}
</style>
