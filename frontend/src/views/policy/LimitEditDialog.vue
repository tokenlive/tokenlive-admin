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
            :label-col="{ style: { width: '96px' } }"
            :wrapper-col="{ flex: 1 }"
            :class="['limit-form', { 'dark-form': appStore.config.theme === 'dark' }]">
            <!-- 规则名称 -->
            <a-form-item
                :label="$t('pages.limit.form.name')"
                name="name">
                <a-input
                    :placeholder="$t('pages.limit.form.name.placeholder')"
                    v-model:value="formData.name"
                    :maxlength="60" />
            </a-form-item>

            <!-- 限流维度 -->
            <a-form-item
                :label="$t('pages.limit.form.type')"
                name="type">
                <a-radio-group v-model:value="formData.type">
                    <a-radio value="request">{{ $t('pages.limit.form.type.request') }}</a-radio>
                    <a-radio value="token">{{ $t('pages.limit.form.type.token') }}</a-radio>
                    <a-radio value="cost">{{ $t('pages.limit.form.type.cost') }}</a-radio>
                </a-radio-group>
            </a-form-item>

            <!-- 估算器配置 (当限流维度为 token 时展示) -->
            <template v-if="formData.type === 'token'">
                <a-form-item
                    :label="$t('pages.limit.form.estimator.type')"
                    name="estimator_type">
                    <a-select
                        v-model:value="formData.estimator.type"
                        style="width: 200px">
                        <a-select-option value="length_ratio">字符比预估 (length_ratio)</a-select-option>
                        <a-select-option value="tiktoken">Tiktoken 计算 (tiktoken)</a-select-option>
                    </a-select>
                </a-form-item>
                <a-form-item
                    :label="$t('pages.limit.form.estimator.ratio')"
                    name="estimator_ratio">
                    <a-input-number
                        v-model:value="formData.estimator.ratio"
                        :min="0.01"
                        :max="10"
                        :step="0.01"
                        style="width: 200px" />
                </a-form-item>
            </template>

            <!-- 流量匹配规则 -->
            <a-form-item
                :label="$t('pages.limit.form.matchMethod')"
                name="relation_type"
                class="match-method-form-item">
                <div class="match-method-section">
                    <div class="match-method-header">
                        <span class="match-method-label">{{ $t('pages.limit.form.matchMethod.settingLabel') }}</span>
                        <a-radio-group v-model:value="formData.relation_type">
                            <a-radio value="AND">AND({{ $t('pages.limit.form.relationType.and') }})</a-radio>
                            <a-radio value="OR">OR({{ $t('pages.limit.form.relationType.or') }})</a-radio>
                        </a-radio-group>
                    </div>

                    <!-- 条件表格 -->
                    <div
                        class="conditions-table"
                        v-if="formData.conditions && formData.conditions.length > 0">
                        <div class="conditions-header">
                            <div class="col-type">{{ $t('pages.limit.form.conditions.type') }}</div>
                            <div class="col-key">{{ $t('pages.limit.form.conditions.key') }}</div>
                            <div class="col-op">
                                {{ $t('pages.limit.form.conditions.opType') }}
                                <a-tooltip :title="$t('pages.limit.form.conditions.opType.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </div>
                            <div class="col-value">{{ $t('pages.limit.form.conditions.values') }}</div>
                            <div class="col-action">{{ $t('pages.limit.form.conditions.action') }}</div>
                        </div>
                        <div
                            class="conditions-row"
                            v-for="(condition, index) in formData.conditions"
                            :key="index">
                            <div class="col-type">
                                <a-select
                                    v-model:value="condition.type"
                                    :placeholder="$t('pages.limit.form.conditions.type.placeholder')"
                                    size="small">
                                    <a-select-option value="header">HEADER</a-select-option>
                                    <a-select-option value="query">QUERY</a-select-option>
                                    <a-select-option value="cookie">COOKIE</a-select-option>
                                    <a-select-option value="system">SYSTEM</a-select-option>
                                    <a-select-option value="tag">TAG</a-select-option>
                                </a-select>
                            </div>
                            <div class="col-key">
                                <a-input
                                    v-model:value="condition.key"
                                    :placeholder="$t('pages.limit.form.conditions.key.placeholder')"
                                    size="small" />
                            </div>
                            <div class="col-op">
                                <a-select
                                    v-model:value="condition.op_type"
                                    :placeholder="$t('pages.limit.form.conditions.opType.placeholder')"
                                    size="small">
                                    <a-select-option value="EQUAL">{{
                                        $t('pages.limit.form.conditions.op.equal')
                                    }}</a-select-option>
                                    <a-select-option value="NOT_EQUAL">{{
                                        $t('pages.limit.form.conditions.op.notEqual')
                                    }}</a-select-option>
                                    <a-select-option value="IN">{{
                                        $t('pages.limit.form.conditions.op.contain')
                                    }}</a-select-option>
                                    <a-select-option value="NOT_IN">{{
                                        $t('pages.limit.form.conditions.op.notContain')
                                    }}</a-select-option>
                                    <a-select-option value="REGULAR">{{
                                        $t('pages.limit.form.conditions.op.regex')
                                    }}</a-select-option>
                                    <a-select-option value="PREFIX">{{
                                        $t('pages.limit.form.conditions.op.prefix')
                                    }}</a-select-option>
                                </a-select>
                            </div>
                            <div class="col-value">
                                <a-select
                                    v-model:value="condition.values"
                                    mode="tags"
                                    :placeholder="$t('pages.limit.form.conditions.values.placeholder')"
                                    size="small" />
                            </div>
                            <div class="col-action">
                                <minus-circle-outlined
                                    class="condition-remove-btn"
                                    @click="removeCondition(index)" />
                            </div>
                        </div>
                    </div>

                    <a
                        class="add-condition-link"
                        @click="addCondition">
                        {{ $t('pages.limit.form.conditions.add') }}
                    </a>
                </div>
            </a-form-item>

            <!-- 配额限制规则 -->
            <a-form-item
                name="sliding_windows"
                class="limit-rules-form-item">
                <template #label>
                    <span>
                        {{ $t('pages.limit.form.limitRules') }}
                        <a-tooltip :title="rulesTooltip">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="limit-rules-section">
                    <div
                        class="limit-rules-table"
                        v-if="formData.sliding_windows && formData.sliding_windows.length > 0">
                        <div class="limit-rules-header">
                            <div class="col-time-window">{{ $t('pages.limit.form.limitRules.timeWindow') }}</div>
                            <div class="col-threshold">
                                <span>{{ thresholdLabel }}</span>
                            </div>
                            <div class="col-burst-ratio">
                                {{ $t('pages.limit.form.limitRules.burstRatio') }}
                            </div>
                            <div class="col-rule-action">{{ $t('pages.limit.form.conditions.action') }}</div>
                        </div>
                        <div
                            class="limit-rules-row"
                            v-for="(window, index) in formData.sliding_windows"
                            :key="index">
                            <div class="col-time-window">
                                <div class="time-window-input">
                                    <a-input-number
                                        v-model:value="window.timeWindowValue"
                                        :placeholder="$t('pages.limit.form.limitRules.timeWindow.placeholder')"
                                        :min="1"
                                        size="small"
                                        style="flex: 1" />
                                    <a-select
                                        v-model:value="window.timeWindowUnit"
                                        size="small"
                                        style="width: 80px">
                                        <a-select-option value="second">{{
                                            $t('pages.limit.form.limitRules.unit.second')
                                        }}</a-select-option>
                                        <a-select-option value="millisecond">{{
                                            $t('pages.limit.form.limitRules.unit.millisecond')
                                        }}</a-select-option>
                                        <a-select-option value="minute">{{
                                            $t('pages.limit.form.limitRules.unit.minute')
                                        }}</a-select-option>
                                    </a-select>
                                </div>
                            </div>
                            <div class="col-threshold">
                                <div class="threshold-input">
                                    <a-input-number
                                        v-model:value="window.threshold"
                                        :placeholder="thresholdPlaceholder"
                                        :min="1"
                                        size="small"
                                        style="flex: 1" />
                                    <span class="threshold-unit">{{ thresholdUnit }}</span>
                                </div>
                            </div>
                            <div class="col-burst-ratio">
                                <div class="burst-ratio-input">
                                    <a-input-number
                                        v-model:value="window.burst_ratio"
                                        placeholder="1.0"
                                        :min="0.01"
                                        :step="0.1"
                                        size="small"
                                        style="width: 100%" />
                                </div>
                            </div>
                            <div class="col-rule-action">
                                <minus-circle-outlined
                                    class="condition-remove-btn"
                                    @click="removeSlidingWindow(index)" />
                            </div>
                        </div>
                    </div>
                    <a
                        class="add-condition-link"
                        @click="addSlidingWindow">
                        {{ $t('pages.limit.form.limitRules.add') }}
                    </a>
                </div>
            </a-form-item>

            <!-- 效果配置（排队等待） -->
            <a-form-item class="limit-scheme-form-item">
                <template #label>
                    <span>
                        {{ $t('pages.limit.form.limitScheme') }}
                        <a-tooltip :title="$t('pages.limit.form.limitScheme.tooltip')">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="limit-scheme-section">
                    <div class="scheme-row">
                        <span class="scheme-label required">{{ $t('pages.limit.form.limitScheme.effect') }}</span>
                        <div class="scheme-buttons">
                            <span
                                v-for="eff in effectOptions"
                                :key="eff.value"
                                :class="['scheme-btn', { active: formData.effect === eff.value }]"
                                @click="formData.effect = eff.value">
                                {{ eff.label }}
                            </span>
                        </div>
                    </div>
                    <!-- 最大排队时长 -->
                    <div
                        class="scheme-row"
                        v-if="formData.effect === 'queuing'">
                        <span class="scheme-label required">{{ $t('pages.limit.form.limitScheme.maxQueueTime') }}</span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.max_queue_time_seconds"
                                :min="0"
                                style="width: 120px" />
                            <span class="scheme-unit">{{ $t('pages.limit.form.limitScheme.maxQueueTime.unit') }}</span>
                        </div>
                    </div>
                </div>
            </a-form-item>

            <!-- 生效状态 -->
            <a-form-item
                :label="$t('pages.limit.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="enabledSwitch"
                    :checked-children="$t('pages.limit.form.enabled.active')"
                    :un-checked-children="$t('pages.limit.form.enabled.inactive')" />
            </a-form-item>

            <!-- 描述 -->
            <a-form-item
                :label="$t('pages.limit.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :placeholder="$t('pages.limit.form.description.placeholder')"
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
import { ref, computed } from 'vue'
import { config } from '@/config'
import apis from '@/apis'
import { useForm, useModal } from '@/hooks'
import { QuestionCircleOutlined, MinusCircleOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'

const appStore = useAppStore()

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

formRules.value = {
    name: { required: true, message: t('pages.limit.form.name.required') },
    type: { required: true, message: t('pages.limit.form.type.required') },
}

const enabledSwitch = computed({
    get: () => formData.value.enabled === 1,
    set: (val) => {
        formData.value.enabled = val ? 1 : 0
    },
})

const effectOptions = computed(() => [
    { value: 'failFast', label: t('pages.limit.form.limitScheme.effect.failFast') },
    { value: 'queuing', label: t('pages.limit.form.limitScheme.effect.queuing') },
])

const thresholdLabel = computed(() => {
    if (formData.value?.type === 'token') {
        return t('pages.limit.form.limitRules.threshold.token')
    } else if (formData.value?.type === 'cost') {
        return t('pages.limit.form.limitRules.threshold.cost')
    }
    return t('pages.limit.form.limitRules.threshold')
})

const thresholdPlaceholder = computed(() => {
    if (formData.value?.type === 'token') {
        return t('pages.limit.form.limitRules.threshold.placeholder.token')
    } else if (formData.value?.type === 'cost') {
        return t('pages.limit.form.limitRules.threshold.placeholder.cost')
    }
    return t('pages.limit.form.limitRules.threshold.placeholder')
})

const thresholdUnit = computed(() => {
    if (formData.value?.type === 'token') {
        return t('pages.limit.form.limitRules.thresholdUnit.token')
    } else if (formData.value?.type === 'cost') {
        return t('pages.limit.form.limitRules.thresholdUnit.cost')
    }
    return t('pages.limit.form.limitRules.thresholdUnit')
})

const rulesTooltip = computed(() => {
    if (formData.value?.type === 'token') {
        return t('pages.limit.form.limitRules.tooltip.token')
    } else if (formData.value?.type === 'cost') {
        return t('pages.limit.form.limitRules.tooltip.cost')
    }
    return t('pages.limit.form.limitRules.tooltip')
})

// ---- 条件行管理 ----
function createEmptyCondition() {
    return { type: undefined, key: '', op_type: undefined, values: [] }
}

function addCondition() {
    if (!formData.value.conditions) {
        formData.value.conditions = []
    }
    formData.value.conditions.push(createEmptyCondition())
}

function removeCondition(index) {
    formData.value.conditions.splice(index, 1)
}

// ---- 滑动窗口管理 ----
function createEmptySlidingWindow() {
    return { threshold: undefined, timeWindowValue: undefined, timeWindowUnit: 'second', burst_ratio: undefined }
}

// 将 timeWindowValue + timeWindowUnit 转换为 timeWindowInMs
function convertToMs(value, unit) {
    if (!value) return 0
    switch (unit) {
        case 'second':
            return value * 1000
        case 'millisecond':
            return value
        case 'minute':
            return value * 60000
        default:
            return value * 1000
    }
}

// 将 timeWindowInMs 转换为 timeWindowValue + timeWindowUnit
function convertFromMs(ms) {
    if (!ms) return { timeWindowValue: undefined, timeWindowUnit: 'second' }
    if (ms % 60000 === 0) return { timeWindowValue: ms / 60000, timeWindowUnit: 'minute' }
    if (ms % 1000 === 0) return { timeWindowValue: ms / 1000, timeWindowUnit: 'second' }
    return { timeWindowValue: ms, timeWindowUnit: 'millisecond' }
}

function addSlidingWindow() {
    if (!formData.value.sliding_windows) {
        formData.value.sliding_windows = []
    }
    formData.value.sliding_windows.push(createEmptySlidingWindow())
}

function removeSlidingWindow(index) {
    formData.value.sliding_windows.splice(index, 1)
}

function handleCreate() {
    formData.value = {
        name: '',
        type: 'request',
        relation_type: 'AND',
        conditions: [createEmptyCondition()],
        sliding_windows: [createEmptySlidingWindow()],
        estimator: { type: 'length_ratio', ratio: 0.25 },
        effect: 'failFast',
        max_queue_time_seconds: 1,
        enabled: 0,
        description: '',
    }
    showModal({
        type: 'create',
        title: t('pages.limit.add'),
    })
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.limit.copy'),
    })

    const { data, success } = await apis.policy.getLimit(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    // 复制模式：不设置 formRecord
    const cloned = cloneDeep(data)
    cloned.name = `${cloned.name} - ${t('pages.policy.copy.suffix')}`
    delete cloned.id
    delete cloned.created_at
    delete cloned.updated_at

    populateFormData(cloned)
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.limit.edit'),
    })

    const { data, success } = await apis.policy.getLimit(record.id).catch()
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
    // 解析 conditions
    if (typeof cloned.conditions === 'string') {
        try {
            cloned.conditions = JSON.parse(cloned.conditions)
        } catch {
            cloned.conditions = []
        }
    }
    if (!Array.isArray(cloned.conditions) || cloned.conditions.length === 0) {
        cloned.conditions = [createEmptyCondition()]
    }

    // 解析 sliding_windows
    if (typeof cloned.sliding_windows === 'string') {
        try {
            cloned.sliding_windows = JSON.parse(cloned.sliding_windows)
        } catch {
            cloned.sliding_windows = []
        }
    }
    if (Array.isArray(cloned.sliding_windows)) {
        cloned.sliding_windows = cloned.sliding_windows.map((w) => {
            const ms = w.time_window_in_ms !== undefined ? w.time_window_in_ms : w.timeWindowInMs
            const { timeWindowValue, timeWindowUnit } = convertFromMs(ms)
            return { ...w, timeWindowValue, timeWindowUnit }
        })
    }
    if (!Array.isArray(cloned.sliding_windows) || cloned.sliding_windows.length === 0) {
        cloned.sliding_windows = [createEmptySlidingWindow()]
    }

    // 解析 estimator
    if (typeof cloned.estimator === 'string') {
        try {
            cloned.estimator = JSON.parse(cloned.estimator)
        } catch {
            cloned.estimator = { type: 'length_ratio', ratio: 0.25 }
        }
    } else if (!cloned.estimator) {
        cloned.estimator = { type: 'length_ratio', ratio: 0.25 }
    }

    cloned.effect = cloned.max_wait_ms > 0 ? 'queuing' : 'failFast'
    cloned.max_queue_time_seconds = cloned.max_wait_ms > 0 ? cloned.max_wait_ms / 1000 : 1

    formData.value = cloned
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async (values) => {
            try {
                showLoading()

                // 将 sliding_windows 转换回 timeWindowInMs 格式
                let slidingWindows = []
                if (formData.value.sliding_windows) {
                    slidingWindows = formData.value.sliding_windows.map((w) => ({
                        threshold: w.threshold,
                        time_window_in_ms: convertToMs(w.timeWindowValue, w.timeWindowUnit),
                        burst_ratio: w.burst_ratio !== undefined && w.burst_ratio !== null ? w.burst_ratio : null,
                    }))
                }

                const maxWaitMs =
                    formData.value.effect === 'queuing' ? (formData.value.max_queue_time_seconds || 0) * 1000 : 0

                const params = {
                    name: values.name,
                    type: values.type,
                    relation_type: formData.value.relation_type,
                    conditions: formData.value.conditions || [],
                    sliding_windows: slidingWindows,
                    estimator: formData.value.type === 'token' ? formData.value.estimator : null,
                    max_wait_ms: maxWaitMs,
                    enabled: formData.value.enabled,
                    description: formData.value.description,
                }

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createLimit(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateLimit(formData.value.id, params).catch(() => {
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
.limit-form {
    :deep(.ant-form-item) {
        margin-bottom: 18px;
    }
}

.form-hint {
    color: #999;
    font-size: 12px;
    line-height: 1.5;
    margin-top: 4px;
}

.match-method-form-item {
    :deep(.ant-form-item-control-input-content) {
        overflow: visible;
    }
}

.match-method-section {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    padding: 16px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.match-method-header {
    display: flex;
    align-items: center;
    margin-bottom: 12px;
}

.match-method-label {
    font-size: 13px;

    margin-right: 8px;
    white-space: nowrap;
}

.conditions-table {
    margin-bottom: 8px;
}

.conditions-header {
    display: flex;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    font-weight: 500;
    font-size: 13px;

    gap: 8px;
}

.conditions-row {
    display: flex;
    align-items: center;
    padding: 8px 0;
    /* removed border-bottom for dark mode */
    gap: 8px;

    &:last-child {
        border-bottom: none;
    }
}

.col-type {
    flex: 0 0 100px;
}

.col-key {
    flex: 0 0 110px;
}

.col-op {
    flex: 0 0 100px;
    display: flex;
    align-items: center;
}

.col-value {
    flex: 1;
    min-width: 0;
}

.col-action {
    flex: 0 0 40px;
    text-align: center;
}

.condition-remove-btn {
    font-size: 18px;
    color: #999;
    cursor: pointer;
    transition: color 0.2s;

    &:hover {
        color: #ff4d4f;
    }
}

.add-condition-link {
    display: inline-block;
    margin-top: 8px;
    color: #1890ff;
    font-size: 13px;
    cursor: pointer;

    &:hover {
        color: #40a9ff;
    }
}

.limit-rules-form-item,
.limit-scheme-form-item {
    :deep(.ant-form-item-control-input-content) {
        overflow: visible;
    }
}

.limit-rules-section {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    padding: 16px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.limit-rules-header {
    display: flex;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    font-weight: 500;
    font-size: 13px;

    gap: 8px;
}

.limit-rules-row {
    display: flex;
    align-items: center;
    padding: 8px 0;
    /* removed border-bottom for dark mode */
    gap: 8px;

    &:last-child {
        border-bottom: none;
    }
}

.col-time-window {
    flex: 1;
}

.col-threshold {
    flex: 1;
}

.col-burst-ratio {
    flex: 0 0 110px;
}

.col-rule-action {
    flex: 0 0 40px;
    text-align: center;
}

.time-window-input,
.threshold-input {
    display: flex;
    align-items: center;
    gap: 4px;
}

.threshold-unit {
    opacity: 0.6;
    font-size: 13px;
    white-space: nowrap;
}

.limit-scheme-section {
    padding: 0;
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
    min-width: 80px;

    &.required::before {
        content: '* ';
        color: #ff4d4f;
    }
}

.scheme-buttons {
    display: flex;
    gap: 0;
    flex-wrap: wrap;
}

.scheme-btn {
    display: inline-block;
    padding: 4px 16px;
    border: 1px solid #d9d9d9;

    cursor: pointer;
    font-size: 13px;

    transition: all 0.2s;
    margin-left: -1px;

    &:first-child {
        border-radius: 4px 0 0 4px;
        margin-left: 0;
    }

    &:last-child {
        border-radius: 0 4px 4px 0;
    }

    &.active {
        color: #1890ff;
        border-color: #1890ff;
        background: #e6f7ff;
        z-index: 1;
        position: relative;
    }

    &:hover:not(.active) {
        color: #1890ff;
        border-color: #1890ff;
        z-index: 1;
        position: relative;
    }
}

.scheme-input {
    display: flex;
    align-items: center;
    gap: 8px;
}

.scheme-unit {
    opacity: 0.6;
    font-size: 13px;
}

.dark-form {
    .match-method-section,
    .limit-rules-section {
        color: rgba(255, 255, 255, 0.85);
        border-color: rgba(255, 255, 255, 0.15);
    }
    .conditions-header,
    .limit-rules-header {
        color: rgba(255, 255, 255, 0.85);
        border-bottom-color: rgba(255, 255, 255, 0.15);
    }
    .match-method-label,
    .scheme-label {
        color: rgba(255, 255, 255, 0.85);
    }
    .threshold-unit,
    .scheme-unit {
        color: rgba(255, 255, 255, 0.45);
    }
    .scheme-btn {
        border-color: rgba(255, 255, 255, 0.15);
        color: rgba(255, 255, 255, 0.85);

        &:hover:not(.active) {
            color: #177ddc;
            border-color: #177ddc;
        }

        &.active {
            background: #111b26;
            color: #177ddc;
            border-color: #177ddc;
        }
    }
}
</style>
