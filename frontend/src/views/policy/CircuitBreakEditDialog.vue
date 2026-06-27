<template>
    <a-drawer
        :open="modal.open"
        :title="modal.title"
        :width="900"
        placement="right"
        :closable="true"
        @close="handleCancel">
        <a-form
            ref="formRef"
            class="circuit-break-form"
            :model="formData"
            :rules="formRules"
            :label-col="{ style: { width: '96px' } }"
            :wrapper-col="{ flex: 1 }">
            <!-- Rule Name -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.name')"
                name="name">
                <a-input
                    :placeholder="$t('pages.circuitBreak.form.name.placeholder')"
                    v-model:value="formData.name"
                    :maxlength="60" />
            </a-form-item>

            <!-- Level -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.level')"
                name="level">
                <a-radio-group
                    v-model:value="formData.level"
                    @change="onLevelChange">
                    <a-radio value="SERVICE">
                        {{ $t('pages.circuitBreak.form.level.service') }}
                    </a-radio>
                    <a-radio value="INSTANCE">
                        {{ $t('pages.circuitBreak.form.level.instance') }}
                    </a-radio>
                </a-radio-group>
            </a-form-item>

            <!-- Trigger Conditions Section -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.triggerConditions')"
                class="section-form-item">
                <template #label>
                    <span>
                        {{ $t('pages.circuitBreak.form.triggerConditions') }}
                        <a-tooltip :title="$t('pages.circuitBreak.form.triggerConditions.tooltip')">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="condition-section">
                    <!-- Check Type -->
                    <div class="condition-row">
                        <span class="condition-label">
                            {{ $t('pages.circuitBreak.form.checkType') }}
                        </span>
                        <a-checkbox-group
                            v-model:value="checkList"
                            @change="onCheckChange">
                            <a-checkbox value="code">
                                {{ $t('pages.circuitBreak.form.checkType.code') }}
                            </a-checkbox>
                            <a-checkbox value="delay">
                                {{ $t('pages.circuitBreak.form.checkType.delay') }}
                            </a-checkbox>
                        </a-checkbox-group>
                    </div>

                    <!-- Error Codes (when code checked) -->
                    <template v-if="checkList.includes('code')">
                        <!-- 💡 提示：错误码与错误消息二选一配置即可，无需同时填写 -->
                        <a-alert
                            type="info"
                            show-icon
                            :message="$t('pages.circuitBreak.form.errorPolicy.tip')"
                            style="margin-bottom: 16px" />

                        <div class="condition-row">
                            <span class="condition-label">
                                {{ $t('pages.circuitBreak.form.errorCodes') }}
                            </span>
                            <a-select
                                v-model:value="formData.error_codes"
                                mode="tags"
                                :placeholder="$t('pages.circuitBreak.form.errorCodes.placeholder')"
                                style="flex: 1" />
                        </div>

                        <!-- 错误消息 -->
                        <div
                            class="condition-row"
                            style="margin-top: 16px">
                            <span class="condition-label">
                                {{ $t('pages.circuitBreak.form.errorMessages') }}
                            </span>
                            <a-select
                                v-model:value="formData.error_messages"
                                mode="tags"
                                :placeholder="$t('pages.circuitBreak.form.errorMessages.placeholder')"
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
                    </template>

                    <!-- Slow Call Duration (when delay checked) -->
                    <template v-if="checkList.includes('delay')">
                        <div
                            v-if="checkList.includes('code')"
                            style="
                                margin-top: 16px;
                                border-top: 1px dashed var(--ant-color-border-secondary, rgba(128, 128, 128, 0.15));
                                padding-top: 16px;
                            " />
                        <div
                            class="condition-row"
                            :style="{ marginTop: checkList.includes('code') ? '0px' : '16px' }">
                            <span class="condition-label required">
                                {{ $t('pages.circuitBreak.form.slowCallDurationThreshold') }}
                            </span>
                            <a-input-number
                                v-model:value="formData.slow_call_duration_threshold"
                                :min="1"
                                style="flex: 1"
                                :placeholder="$t('pages.circuitBreak.form.slowCallDurationThreshold.placeholder')" />
                            <span class="condition-unit">
                                {{ $t('pages.circuitBreak.form.slowCallDurationThreshold.unit') }}
                            </span>
                        </div>
                        <div class="condition-row">
                            <span class="condition-label required">
                                {{ $t('pages.circuitBreak.form.slowCallMetric') }}
                            </span>
                            <a-select
                                v-model:value="formData.slow_call_metric"
                                :placeholder="$t('pages.circuitBreak.form.slowCallMetric.placeholder')"
                                style="flex: 1">
                                <a-select-option value="TTFT">{{
                                    $t('pages.circuitBreak.form.slowCallMetric.ttft')
                                }}</a-select-option>
                                <a-select-option value="RTT">{{
                                    $t('pages.circuitBreak.form.slowCallMetric.duration')
                                }}</a-select-option>
                            </a-select>
                        </div>
                    </template>
                </div>
            </a-form-item>

            <!-- Judgment Conditions -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.judgment')"
                class="section-form-item">
                <template #label>
                    <span>
                        {{ $t('pages.circuitBreak.form.judgment') }}
                        <a-tooltip :title="$t('pages.circuitBreak.form.judgment.tooltip')">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="condition-section">
                    <div
                        v-for="(item, index) in formData.judgment"
                        :key="index"
                        class="judgment-row">
                        <a-select
                            v-model:value="item.key"
                            :placeholder="$t('pages.circuitBreak.form.judgment.key.placeholder')"
                            style="width: 200px">
                            <a-select-option
                                v-if="checkList.includes('code')"
                                value="failureRateThreshold">
                                {{ $t('pages.circuitBreak.form.judgment.failureRate') }}
                            </a-select-option>
                            <a-select-option
                                v-if="checkList.includes('delay')"
                                value="slowCallRateThreshold">
                                {{ $t('pages.circuitBreak.form.judgment.slowCallRate') }}
                            </a-select-option>
                        </a-select>
                        <span
                            class="condition-unit"
                            style="margin: 0 8px"
                            >&ge;</span
                        >
                        <a-input-number
                            v-model:value="item.value"
                            :min="1"
                            :max="100"
                            style="width: 120px"
                            :placeholder="$t('pages.circuitBreak.form.judgment.value.placeholder')" />
                        <span class="condition-unit">%</span>
                        <a-button
                            v-if="formData.judgment.length > 1"
                            type="link"
                            size="small"
                            danger
                            @click="removeJudgment(index)">
                            <delete-outlined />
                        </a-button>
                    </div>
                    <a-button
                        v-if="availableJudgmentCount > 0"
                        type="dashed"
                        block
                        size="small"
                        @click="addJudgment"
                        style="margin-top: 8px">
                        <plus-outlined />
                        {{ $t('pages.circuitBreak.form.judgment.add') }}
                    </a-button>
                </div>
            </a-form-item>

            <!-- Sliding Window Config -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.slidingWindow')"
                class="section-form-item">
                <div class="condition-section">
                    <div class="scheme-row">
                        <span class="scheme-label required">
                            {{ $t('pages.circuitBreak.form.slidingWindowType') }}
                        </span>
                        <a-radio-group
                            v-model:value="formData.sliding_window_type"
                            @change="onWindowTypeChange">
                            <a-radio value="time">
                                {{ $t('pages.circuitBreak.form.slidingWindowType.time') }}
                            </a-radio>
                            <a-radio value="count">
                                {{ $t('pages.circuitBreak.form.slidingWindowType.count') }}
                            </a-radio>
                        </a-radio-group>
                    </div>
                    <div class="scheme-row">
                        <span class="scheme-label required">
                            {{
                                formData.sliding_window_type === 'time'
                                    ? $t('pages.circuitBreak.form.slidingWindow.timeWindow')
                                    : $t('pages.circuitBreak.form.slidingWindow.countWindow')
                            }}
                        </span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.sliding_window_size"
                                :min="1"
                                style="width: 100%"
                                :placeholder="$t('pages.circuitBreak.form.slidingWindow.size.placeholder')" />
                            <span class="scheme-unit">
                                {{
                                    formData.sliding_window_type === 'time'
                                        ? $t('pages.circuitBreak.form.slidingWindow.unit.second')
                                        : $t('pages.circuitBreak.form.slidingWindow.unit.count')
                                }}
                            </span>
                        </div>
                    </div>
                    <div class="scheme-row">
                        <span class="scheme-label required">
                            {{ $t('pages.circuitBreak.form.minCallsThreshold') }}
                        </span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.min_calls_threshold"
                                :min="1"
                                style="width: 100%"
                                :placeholder="$t('pages.circuitBreak.form.minCallsThreshold.placeholder')" />
                            <span class="scheme-unit">
                                {{ $t('pages.circuitBreak.form.minCallsThreshold.unit') }}
                            </span>
                        </div>
                    </div>
                </div>
            </a-form-item>

            <!-- Open State Config -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.openState')"
                class="section-form-item">
                <div class="condition-section">
                    <div class="scheme-row">
                        <span class="scheme-label required">
                            {{ $t('pages.circuitBreak.form.waitDurationInOpenState') }}
                        </span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.wait_duration_in_open_state"
                                :min="1"
                                style="width: 100%"
                                :placeholder="$t('pages.circuitBreak.form.waitDurationInOpenState.placeholder')" />
                            <span class="scheme-unit">
                                {{ $t('pages.circuitBreak.form.waitDurationInOpenState.unit') }}
                            </span>
                        </div>
                    </div>
                    <div class="scheme-row">
                        <span class="scheme-label required">
                            {{ $t('pages.circuitBreak.form.allowedCallsInHalfOpenState') }}
                        </span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.allowed_calls_in_half_open_state"
                                :min="1"
                                style="width: 100%"
                                :placeholder="$t('pages.circuitBreak.form.allowedCallsInHalfOpenState.placeholder')" />
                            <span class="scheme-unit">
                                {{ $t('pages.circuitBreak.form.allowedCallsInHalfOpenState.unit') }}
                            </span>
                        </div>
                    </div>
                    <!-- Outlier Max Percent (Only for INSTANCE) -->
                    <div
                        class="scheme-row"
                        v-if="formData.level === 'INSTANCE'">
                        <span class="scheme-label required">
                            {{ $t('pages.circuitBreak.form.outlierMaxPercent') }}
                        </span>
                        <div class="scheme-input">
                            <a-input-number
                                v-model:value="formData.outlier_max_percent"
                                :min="1"
                                :max="100"
                                style="width: 100%"
                                :placeholder="$t('pages.circuitBreak.form.outlierMaxPercent.placeholder')" />
                            <span class="scheme-unit">%</span>
                        </div>
                    </div>
                    <!-- Force Open -->
                    <div class="scheme-row">
                        <span class="scheme-label">
                            {{ $t('pages.circuitBreak.form.forceOpen') }}
                        </span>
                        <a-switch
                            v-model:checked="forceOpenSwitch"
                            :checked-children="$t('pages.circuitBreak.form.forceOpen.yes')"
                            :un-checked-children="$t('pages.circuitBreak.form.forceOpen.no')" />
                    </div>
                </div>
            </a-form-item>

            <!-- Degrade Config (hidden for INSTANCE) -->
            <template v-if="formData.level !== 'INSTANCE'">
                <a-form-item
                    :label="$t('pages.circuitBreak.form.degradeConfig')"
                    class="section-form-item">
                    <div class="condition-section">
                        <div class="scheme-row">
                            <span class="scheme-label">
                                {{ $t('pages.circuitBreak.form.responseCode') }}
                            </span>
                            <a-input
                                v-model:value="formData.responseCode"
                                :placeholder="$t('pages.circuitBreak.form.responseCode.placeholder')"
                                style="flex: 1" />
                        </div>
                        <div class="scheme-row">
                            <span class="scheme-label">
                                {{ $t('pages.circuitBreak.form.responseHeaders') }}
                            </span>
                            <div style="flex: 1">
                                <div
                                    v-for="(item, index) in formData.attributes"
                                    :key="index"
                                    class="attribute-row">
                                    <a-input
                                        v-model:value="item.key"
                                        :placeholder="$t('pages.circuitBreak.form.responseHeaders.key.placeholder')"
                                        style="flex: 1" />
                                    <a-input
                                        v-model:value="item.value"
                                        :placeholder="$t('pages.circuitBreak.form.responseHeaders.value.placeholder')"
                                        style="flex: 1; margin-left: 8px" />
                                    <a-button
                                        v-if="formData.attributes.length > 1"
                                        type="link"
                                        size="small"
                                        danger
                                        @click="removeAttribute(index)">
                                        <delete-outlined />
                                    </a-button>
                                    <a-button
                                        v-if="
                                            index === formData.attributes.length - 1 && formData.attributes.length < 10
                                        "
                                        type="link"
                                        size="small"
                                        @click="addAttribute">
                                        <plus-outlined />
                                    </a-button>
                                </div>
                            </div>
                        </div>
                        <div class="scheme-row">
                            <span class="scheme-label">
                                {{ $t('pages.circuitBreak.form.responseBody') }}
                            </span>
                            <a-textarea
                                v-model:value="formData.responseBody"
                                :placeholder="$t('pages.circuitBreak.form.responseBody.placeholder')"
                                :rows="3"
                                style="flex: 1" />
                        </div>
                    </div>
                </a-form-item>
            </template>

            <!-- Enabled -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.enabled')"
                name="enabled">
                <a-switch
                    v-model:checked="enabledSwitch"
                    :checked-children="$t('pages.circuitBreak.form.enabled.active')"
                    :un-checked-children="$t('pages.circuitBreak.form.enabled.inactive')" />
            </a-form-item>

            <!-- Description -->
            <a-form-item
                :label="$t('pages.circuitBreak.form.description')"
                name="description">
                <a-textarea
                    v-model:value="formData.description"
                    :placeholder="$t('pages.circuitBreak.form.description.placeholder')"
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
import { QuestionCircleOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useI18n } from 'vue-i18n'

const emit = defineEmits(['ok'])
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))

const checkList = ref(['code'])

formRules.value = {
    name: { required: true, message: t('pages.circuitBreak.form.name.required') },
    level: { required: true, message: t('pages.circuitBreak.form.level.required') },
    sliding_window_type: {
        required: true,
        message: t('pages.circuitBreak.form.slidingWindowType.required'),
    },
    sliding_window_size: {
        required: true,
        message: t('pages.circuitBreak.form.slidingWindow.size.required'),
    },
    min_calls_threshold: {
        required: true,
        message: t('pages.circuitBreak.form.minCallsThreshold.required'),
    },
    wait_duration_in_open_state: {
        required: true,
        message: t('pages.circuitBreak.form.waitDurationInOpenState.required'),
    },
    allowed_calls_in_half_open_state: {
        required: true,
        message: t('pages.circuitBreak.form.allowedCallsInHalfOpenState.required'),
    },
}

const enabledSwitch = computed({
    get: () => formData.value.enabled === 1,
    set: (val) => {
        formData.value.enabled = val ? 1 : 0
    },
})

const forceOpenSwitch = computed({
    get: () => formData.value.force_open === 1,
    set: (val) => {
        formData.value.force_open = val ? 1 : 0
    },
})

const availableJudgmentCount = computed(() => {
    const allKeys = []
    if (checkList.value.includes('code')) allKeys.push('failureRateThreshold')
    if (checkList.value.includes('delay')) allKeys.push('slowCallRateThreshold')
    const selectedKeys = formData.value.judgment.map((item) => item.key).filter(Boolean)
    return allKeys.length - selectedKeys.length
})

function onLevelChange() {
    formData.value.force_open = 0
    formData.value.responseCode = '200'
    formData.value.attributes = [{ key: '', value: '' }]
    formData.value.responseBody = ''
}

function onCheckChange() {
    if (!checkList.value.includes('code')) {
        formData.value.error_codes = []
        formData.value.error_messages = []
        formData.value.codePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
        formData.value.messagePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
        // Remove failureRateThreshold from judgment
        formData.value.judgment = formData.value.judgment.filter((item) => item.key !== 'failureRateThreshold')
    }
    if (!checkList.value.includes('delay')) {
        formData.value.slow_call_duration_threshold = undefined
        formData.value.slow_call_metric = ''
        // Remove slowCallRateThreshold from judgment
        formData.value.judgment = formData.value.judgment.filter((item) => item.key !== 'slowCallRateThreshold')
    }
    // Ensure at least one empty judgment row remains
    if (formData.value.judgment.length === 0) {
        formData.value.judgment.push({ key: '', value: undefined })
    }
}

function onWindowTypeChange() {
    formData.value.sliding_window_size = undefined
    formData.value.min_calls_threshold = undefined
}

function addJudgment() {
    formData.value.judgment.push({ key: '', value: undefined })
}

function removeJudgment(index) {
    formData.value.judgment.splice(index, 1)
}

function addAttribute() {
    formData.value.attributes.push({ key: '', value: '' })
}

function removeAttribute(index) {
    formData.value.attributes.splice(index, 1)
}

function handleCreate() {
    formData.value.enabled = 0
    formData.value.level = 'SERVICE'
    formData.value.sliding_window_type = 'time'
    formData.value.force_open = 0
    formData.value.outlier_max_percent = 10
    formData.value.failure_rate_threshold = 1
    formData.value.slow_call_rate_threshold = 1
    formData.value.slow_call_metric = ''
    formData.value.error_codes = []
    formData.value.error_messages = []
    formData.value.codePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    formData.value.messagePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    formData.value.judgment = [{ key: 'failureRateThreshold', value: undefined }]
    formData.value.attributes = [{ key: '', value: '' }]
    formData.value.responseCode = '200'
    formData.value.responseBody = ''
    checkList.value = ['code']
    showModal({
        type: 'create',
        title: t('pages.circuitBreak.add'),
    })
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.circuitBreak.copy'),
    })

    const { data, success } = await apis.policy.getCircuitBreak(record.id).catch()
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
        title: t('pages.circuitBreak.edit'),
    })

    const { data, success } = await apis.policy.getCircuitBreak(record.id).catch()
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
    // Parse JSON fields from backend
    cloned.error_codes = cloned.error_codes || []
    cloned.error_messages = cloned.error_messages || []

    // Parse code_policy
    const rawCodePolicy = cloned.code_policy || cloned.codePolicy
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

    // Parse message_policy
    const rawMessagePolicy = cloned.message_policy || cloned.messagePolicy
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

    // Set slow_call_metric default
    if (!(cloned.slow_call_duration_threshold > 0 || cloned.slow_call_rate_threshold > 0)) {
        cloned.slow_call_metric = ''
    }

    // Build judgment from thresholds
    cloned.judgment = []
    if (cloned.failure_rate_threshold > 0) {
        cloned.judgment.push({
            key: 'failureRateThreshold',
            value: cloned.failure_rate_threshold,
        })
    }
    if (cloned.slow_call_rate_threshold > 0) {
        cloned.judgment.push({
            key: 'slowCallRateThreshold',
            value: cloned.slow_call_rate_threshold,
        })
    }
    if (cloned.judgment.length === 0) {
        cloned.judgment.push({ key: '', value: undefined })
    }

    // Parse degrade_config
    if (cloned.degrade_config) {
        cloned.responseCode = String(cloned.degrade_config.response_code)
        cloned.responseBody = cloned.degrade_config.response_body
        const attrs = cloned.degrade_config.attributes || {}
        cloned.attributes = Object.entries(attrs).map(([key, value]) => ({
            key,
            value,
        }))
        if (cloned.attributes.length === 0) {
            cloned.attributes.push({ key: '', value: '' })
        }
    } else {
        cloned.responseCode = '200'
        cloned.responseBody = ''
        cloned.attributes = [{ key: '', value: '' }]
    }

    // Determine checkList
    checkList.value = []
    if (
        cloned.error_codes?.length > 0 ||
        cloned.error_messages?.length > 0 ||
        cloned.codePolicy.parser ||
        cloned.messagePolicy.parser
    ) {
        checkList.value.push('code')
    }
    if (cloned.slow_call_duration_threshold > 0) {
        checkList.value.push('delay')
    }
    if (checkList.value.length === 0) {
        checkList.value.push('code')
    }

    cloned.outlier_max_percent = cloned.outlier_max_percent !== undefined ? cloned.outlier_max_percent : 10
    formData.value = cloned
}

function handleOk() {
    formRef.value
        .validateFields()
        .then(async () => {
            try {
                showLoading()

                // Build judgment -> thresholds
                let failureRateThreshold = 0
                let slowCallRateThreshold = 0
                // Only include thresholds if corresponding check is enabled
                if (checkList.value.includes('code')) {
                    for (const item of formData.value.judgment) {
                        if (item.key === 'failureRateThreshold' && item.value) {
                            failureRateThreshold = item.value
                        }
                    }
                }
                if (checkList.value.includes('delay')) {
                    for (const item of formData.value.judgment) {
                        if (item.key === 'slowCallRateThreshold' && item.value) {
                            slowCallRateThreshold = item.value
                        }
                    }
                }

                // Build code_policy - only if code check is enabled
                let codePolicy = undefined
                if (checkList.value.includes('code') && formData.value.codePolicy && formData.value.codePolicy.parser) {
                    codePolicy = {
                        parser: formData.value.codePolicy.parser,
                        expression: formData.value.codePolicy.expression,
                        statuses: formData.value.codePolicy.statuses || [],
                        content_types: formData.value.codePolicy.contentTypes || [],
                    }
                }

                // Build message_policy - only if code check is enabled
                let messagePolicy = undefined
                if (
                    checkList.value.includes('code') &&
                    formData.value.messagePolicy &&
                    formData.value.messagePolicy.parser
                ) {
                    messagePolicy = {
                        parser: formData.value.messagePolicy.parser,
                        expression: formData.value.messagePolicy.expression,
                        statuses: formData.value.messagePolicy.statuses || [],
                        content_types: formData.value.messagePolicy.contentTypes || [],
                    }
                }

                // Build slow call fields - only if delay check is enabled
                let slowCallMetric = ''
                let slowCallDurationThreshold = 0
                if (checkList.value.includes('delay')) {
                    slowCallDurationThreshold = formData.value.slow_call_duration_threshold || 0
                    if (slowCallRateThreshold > 0 || slowCallDurationThreshold > 0) {
                        slowCallMetric = formData.value.slow_call_metric
                    }
                }

                // Build degrade_config (not for INSTANCE)
                let degradeConfig = undefined
                if (formData.value.level !== 'INSTANCE') {
                    const attrs = {}
                    for (const item of formData.value.attributes) {
                        if (item.key && item.value) {
                            attrs[item.key] = item.value
                        }
                    }
                    degradeConfig = {
                        response_code: parseInt(formData.value.responseCode, 10) || 200,
                        attributes: attrs,
                        response_body: formData.value.responseBody,
                    }
                }

                const params = {
                    name: formData.value.name,
                    level: formData.value.level,
                    sliding_window_type: formData.value.sliding_window_type,
                    sliding_window_size: formData.value.sliding_window_size,
                    min_calls_threshold: formData.value.min_calls_threshold,
                    code_policy: codePolicy,
                    message_policy: messagePolicy,
                    error_codes: checkList.value.includes('code') ? formData.value.error_codes || [] : [],
                    error_messages: checkList.value.includes('code') ? formData.value.error_messages || [] : [],
                    failure_rate_threshold: failureRateThreshold,
                    slow_call_rate_threshold: slowCallRateThreshold,
                    slow_call_duration_threshold: slowCallDurationThreshold,
                    slow_call_metric: slowCallMetric,
                    wait_duration_in_open_state: formData.value.wait_duration_in_open_state,
                    allowed_calls_in_half_open_state: formData.value.allowed_calls_in_half_open_state,
                    force_open: formData.value.force_open || 0,
                    outlier_max_percent:
                        formData.value.level === 'INSTANCE' ? formData.value.outlier_max_percent || 10 : 0,
                    degrade_config: degradeConfig,
                    enabled: formData.value.enabled,
                    description: formData.value.description,
                }

                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createCircuitBreak(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateCircuitBreak(formData.value.id, params).catch(() => {
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
    checkList.value = ['code']
    formData.value.codePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
    formData.value.messagePolicy = { parser: '', expression: '', statuses: [], contentTypes: ['application/json'] }
}

defineExpose({
    handleCreate,
    handleEdit,
    handleCopy,
})
</script>

<style lang="less" scoped>
.circuit-break-form {
    :deep(.ant-form-item) {
        margin-bottom: 18px;
    }
}

.section-form-item {
    :deep(.ant-form-item-control-input-content) {
        overflow: visible;
    }
}

.condition-section {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    padding: 16px;
}

.condition-row {
    display: flex;
    align-items: center;
    margin-bottom: 12px;
    gap: 8px;

    &:last-child {
        margin-bottom: 0;
    }
}

.condition-label {
    font-size: 13px;

    white-space: nowrap;
    min-width: 70px;
    display: flex;
    align-items: center;

    &.required::before {
        content: '* ';
        color: #ff4d4f;
    }
}

.condition-prefix {
    font-size: 13px;
    opacity: 0.6;
    white-space: nowrap;
}

.condition-suffix {
    font-size: 13px;
    opacity: 0.6;
    white-space: nowrap;
}

.condition-unit {
    font-size: 13px;
    opacity: 0.6;
    white-space: nowrap;
}

.condition-jsonpath {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
    opacity: 0.6;
    white-space: nowrap;
}

.jsonpath-label {
    font-size: 13px;
    opacity: 0.6;
}

.jsonpath-input {
    width: 120px;
}

.judgment-row {
    display: flex;
    align-items: center;
    margin-bottom: 8px;

    &:last-of-type {
        margin-bottom: 0;
    }
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

.attribute-row {
    display: flex;
    align-items: center;
    margin-bottom: 8px;

    &:last-of-type {
        margin-bottom: 0;
    }
}
</style>
