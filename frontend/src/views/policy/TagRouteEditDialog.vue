<template>
    <a-drawer
        :open="modal.open"
        :title="modal.title"
        :width="900"
        placement="right"
        :closable="true"
        @close="handleCancel">
        <div :class="{ 'dark-form': appStore.config.theme === 'dark' }">
            <a-form
                ref="formRef"
                :model="formData"
                :rules="formRules"
                :label-col="{ style: { width: '140px' } }"
                :wrapper-col="{ flex: 1 }">
                <!-- 规则名称 -->
                <a-form-item
                    :label="$t('pages.tagRoute.form.name')"
                    name="name">
                    <a-input
                        :placeholder="$t('pages.tagRoute.form.name.placeholder')"
                        v-model:value="formData.name"
                        :maxlength="60" />
                </a-form-item>

                <!-- 生效状态 -->
                <a-form-item
                    :label="$t('pages.tagRoute.form.enabled')"
                    name="enabled">
                    <a-switch
                        v-model:checked="enabledSwitch"
                        :checked-children="$t('pages.tagRoute.form.enabled.active')"
                        :un-checked-children="$t('pages.tagRoute.form.enabled.inactive')" />
                </a-form-item>

                <!-- 描述 -->
                <a-form-item
                    :label="$t('pages.tagRoute.form.description')"
                    name="description">
                    <a-textarea
                        v-model:value="formData.description"
                        :placeholder="$t('pages.tagRoute.form.description.placeholder')"
                        :maxlength="255"
                        show-count
                        :rows="3" />
                </a-form-item>
            </a-form>

            <!-- 路由规则列表 (RouteDetail 1:n) -->
            <div class="details-section">
                <div
                    class="rule-card"
                    v-for="(detail, rIndex) in formData.details"
                    :key="String(rIndex)">
                    <div class="rule-card-header">
                        <span class="rule-card-title">{{ $t('pages.tagRoute.form.details.rule') + (rIndex + 1) }}</span>
                        <close-outlined
                            class="rule-remove-btn"
                            @click.stop="removeDetail(rIndex)" />
                    </div>
                    <div class="rule-card-body">
                        <!-- 匹配条件 -->
                        <div class="rule-field">
                            <div class="rule-field-label required">
                                {{ $t('pages.tagRoute.form.matchMethod.settingLabel') }}
                            </div>
                            <div class="rule-field-content">
                                <div class="relation-row">
                                    <span>{{ $t('pages.tagRoute.form.matchMethod.settingLabel') }}：</span>
                                    <a-radio-group
                                        v-model:value="detail.relation_type"
                                        size="small">
                                        <a-radio value="AND"
                                            >AND({{ $t('pages.tagRoute.form.relationType.and') }})</a-radio
                                        >
                                        <a-radio value="OR"
                                            >OR({{ $t('pages.tagRoute.form.relationType.or') }})</a-radio
                                        >
                                    </a-radio-group>
                                </div>
                                <div
                                    class="rule-table"
                                    v-if="detail.conditions && detail.conditions.length > 0">
                                    <div class="rule-table-header">
                                        <div class="col-type">{{ $t('pages.tagRoute.form.conditions.type') }}</div>
                                        <div class="col-key">{{ $t('pages.tagRoute.form.conditions.key') }}</div>
                                        <div class="col-op">
                                            {{ $t('pages.tagRoute.form.conditions.opType') }}
                                            <a-tooltip :title="$t('pages.tagRoute.form.conditions.opType.tooltip')">
                                                <question-circle-outlined style="margin-left: 4px; color: #999" />
                                            </a-tooltip>
                                        </div>
                                        <div class="col-value">{{ $t('pages.tagRoute.form.conditions.values') }}</div>
                                        <div class="col-action">{{ $t('pages.tagRoute.form.conditions.action') }}</div>
                                    </div>
                                    <div
                                        class="rule-table-row"
                                        v-for="(condition, cIndex) in detail.conditions"
                                        :key="cIndex">
                                        <div class="col-type">
                                            <a-select
                                                v-model:value="condition.type"
                                                :placeholder="$t('pages.tagRoute.form.conditions.type.placeholder')"
                                                size="small"
                                                style="width: 100%">
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
                                                :placeholder="$t('pages.tagRoute.form.conditions.key.placeholder')"
                                                size="small"
                                                style="width: 100%" />
                                        </div>
                                        <div class="col-op">
                                            <a-select
                                                v-model:value="condition.op_type"
                                                :placeholder="$t('pages.tagRoute.form.conditions.opType.placeholder')"
                                                size="small"
                                                style="width: 100%">
                                                <a-select-option value="EQUAL">{{
                                                    $t('pages.tagRoute.form.conditions.op.equal')
                                                }}</a-select-option>
                                                <a-select-option value="NOT_EQUAL">{{
                                                    $t('pages.tagRoute.form.conditions.op.notEqual')
                                                }}</a-select-option>
                                                <a-select-option value="IN">{{
                                                    $t('pages.tagRoute.form.conditions.op.contain')
                                                }}</a-select-option>
                                                <a-select-option value="NOT_IN">{{
                                                    $t('pages.tagRoute.form.conditions.op.notContain')
                                                }}</a-select-option>
                                                <a-select-option value="REGULAR">{{
                                                    $t('pages.tagRoute.form.conditions.op.regex')
                                                }}</a-select-option>
                                                <a-select-option value="PREFIX">{{
                                                    $t('pages.tagRoute.form.conditions.op.prefix')
                                                }}</a-select-option>
                                            </a-select>
                                        </div>
                                        <div class="col-value">
                                            <a-select
                                                v-model:value="condition.values"
                                                mode="tags"
                                                :placeholder="$t('pages.tagRoute.form.conditions.values.placeholder')"
                                                size="small"
                                                style="width: 100%" />
                                        </div>
                                        <div class="col-action">
                                            <minus-circle-outlined
                                                class="icon-btn"
                                                @click="removeCondition(rIndex, cIndex)" />
                                            <plus-circle-outlined
                                                class="icon-btn"
                                                @click="addCondition(rIndex)" />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- 目的地标签 -->
                        <div class="rule-field">
                            <div class="rule-field-label required">
                                {{ $t('pages.tagRoute.form.destinations') }}
                                <a-tooltip
                                    :title="
                                        $t('pages.tagRoute.form.destinations.tooltip') ||
                                        '目地标签规则中若有多个键值，默认采用AND逻辑关系'
                                    ">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </div>
                            <div class="rule-field-content">
                                <div
                                    class="dest-item"
                                    v-for="(destination, dIndex) in detail.destinations"
                                    :key="dIndex">
                                    <div class="dest-box">
                                        <div
                                            class="dest-conditions-table"
                                            v-if="destination.conditions && destination.conditions.length > 0">
                                            <div
                                                class="dest-condition-row"
                                                v-for="(dc, dcIndex) in destination.conditions"
                                                :key="dcIndex">
                                                <div class="dest-col-key">
                                                    <a-input
                                                        v-model:value="dc.key"
                                                        :placeholder="
                                                            $t('pages.tagRoute.form.conditions.key.placeholder')
                                                        "
                                                        size="small"
                                                        style="width: 100%" />
                                                </div>
                                                <div class="dest-col-op">
                                                    <a-select
                                                        v-model:value="dc.op_type"
                                                        :placeholder="
                                                            $t('pages.tagRoute.form.conditions.opType.placeholder')
                                                        "
                                                        size="small"
                                                        style="width: 100%">
                                                        <a-select-option value="EQUAL">{{
                                                            $t('pages.tagRoute.form.conditions.op.equal')
                                                        }}</a-select-option>
                                                        <a-select-option value="NOT_EQUAL">{{
                                                            $t('pages.tagRoute.form.conditions.op.notEqual')
                                                        }}</a-select-option>
                                                        <a-select-option value="IN">{{
                                                            $t('pages.tagRoute.form.conditions.op.contain')
                                                        }}</a-select-option>
                                                        <a-select-option value="NOT_IN">{{
                                                            $t('pages.tagRoute.form.conditions.op.notContain')
                                                        }}</a-select-option>
                                                        <a-select-option value="REGULAR">{{
                                                            $t('pages.tagRoute.form.conditions.op.regex')
                                                        }}</a-select-option>
                                                        <a-select-option value="PREFIX">{{
                                                            $t('pages.tagRoute.form.conditions.op.prefix')
                                                        }}</a-select-option>
                                                    </a-select>
                                                </div>
                                                <div class="dest-col-value">
                                                    <a-select
                                                        v-model:value="dc.values"
                                                        mode="tags"
                                                        :placeholder="
                                                            $t('pages.tagRoute.form.conditions.values.placeholder')
                                                        "
                                                        size="small"
                                                        style="width: 100%" />
                                                </div>
                                                <div class="dest-col-action">
                                                    <minus-circle-outlined
                                                        class="icon-btn"
                                                        @click="removeDestCondition(rIndex, dIndex, dcIndex)" />
                                                    <plus-circle-outlined
                                                        class="icon-btn"
                                                        @click="addDestCondition(rIndex, dIndex)" />
                                                </div>
                                            </div>
                                        </div>
                                        <div class="dest-weight-row">
                                            <span
                                                class="required"
                                                style="margin-right: 8px">
                                                {{ $t('pages.tagRoute.form.destinations.weight') }}
                                                <a-tooltip
                                                    :title="$t('pages.tagRoute.form.destinations.weight.tooltip')">
                                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                                </a-tooltip>
                                            </span>
                                            <a-input-number
                                                v-model:value="destination.weight"
                                                :min="0"
                                                size="small"
                                                style="width: 150px" />
                                        </div>
                                    </div>
                                    <div class="dest-outer-action">
                                        <minus-circle-outlined
                                            class="icon-btn"
                                            @click="removeDestination(rIndex, dIndex)" />
                                        <plus-circle-outlined
                                            class="icon-btn"
                                            @click="addDestination(rIndex)" />
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- 优先级 -->
                        <div class="rule-field">
                            <div class="rule-field-label required">
                                {{ $t('pages.tagRoute.form.order') }}
                                <a-tooltip :title="$t('pages.tagRoute.form.order.tooltip')">
                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                </a-tooltip>
                            </div>
                            <div class="rule-field-content">
                                <a-input-number
                                    v-model:value="detail.order"
                                    :min="1"
                                    size="small"
                                    style="width: 150px" />
                            </div>
                        </div>
                    </div>
                </div>

                <a-button
                    type="link"
                    size="small"
                    @click="addDetail"
                    class="add-rule-btn">
                    <template #icon><plus-outlined /></template>
                    {{ $t('pages.tagRoute.form.details.add') }}
                </a-button>
            </div>
        </div>

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
import { useAppStore } from '@/store'

const appStore = useAppStore()
import {
    QuestionCircleOutlined,
    MinusCircleOutlined,
    PlusOutlined,
    CloseOutlined,
    PlusCircleOutlined,
} from '@ant-design/icons-vue'

const emit = defineEmits(['ok'])
import { useI18n } from 'vue-i18n'
const { modal, showModal, hideModal, showLoading, hideLoading } = useModal()
const { formRecord, formData, formRef, formRules, resetForm } = useForm()
const { t } = useI18n()
const cancelText = ref(t('button.cancel'))
const okText = ref(t('button.confirm'))
const activeDetailKeys = ref([])

formRules.value = {
    name: { required: true, message: t('pages.tagRoute.form.name.required') },
}

const enabledSwitch = computed({
    get: () => formData.value.enabled === 1,
    set: (val) => {
        formData.value.enabled = val ? 1 : 0
    },
})

// ---- Detail (RouteDetail) 管理 ----
function createEmptyDetail() {
    return {
        _id: undefined,
        order: 0,
        relation_type: 'AND',
        conditions: [createEmptyCondition()],
        destinations: [createEmptyDestination()],
    }
}

function addDetail() {
    if (!formData.value.details) {
        formData.value.details = []
    }
    const idx = formData.value.details.length
    formData.value.details.push(createEmptyDetail())
    activeDetailKeys.value.push(String(idx))
}

function removeDetail(index) {
    formData.value.details.splice(index, 1)
}

// ---- 条件行管理 ----
function createEmptyCondition() {
    return { type: undefined, key: '', op_type: undefined, values: [] }
}

function addCondition(rIndex) {
    if (!formData.value.details[rIndex].conditions) {
        formData.value.details[rIndex].conditions = []
    }
    formData.value.details[rIndex].conditions.push(createEmptyCondition())
}

function removeCondition(rIndex, cIndex) {
    formData.value.details[rIndex].conditions.splice(cIndex, 1)
}

// ---- 目标地址管理 ----
function createEmptyDestination() {
    return {
        weight: 100,
        relation_type: 'AND',
        conditions: [createEmptyDestCondition()],
    }
}

function createEmptyDestCondition() {
    return { op_type: undefined, key: '', values: [] }
}

function addDestination(rIndex) {
    if (!formData.value.details[rIndex].destinations) {
        formData.value.details[rIndex].destinations = []
    }
    formData.value.details[rIndex].destinations.push(createEmptyDestination())
}

function removeDestination(rIndex, dIndex) {
    formData.value.details[rIndex].destinations.splice(dIndex, 1)
}

function addDestCondition(rIndex, dIndex) {
    if (!formData.value.details[rIndex].destinations[dIndex].conditions) {
        formData.value.details[rIndex].destinations[dIndex].conditions = []
    }
    formData.value.details[rIndex].destinations[dIndex].conditions.push(createEmptyDestCondition())
}

function removeDestCondition(rIndex, dIndex, dcIndex) {
    formData.value.details[rIndex].destinations[dIndex].conditions.splice(dcIndex, 1)
}

// ---- 数据加载 ----
async function loadRouteDetails(routeId) {
    try {
        const { success, data } = await apis.policy
            .getRouteDetailList({ route_id: routeId, pageSize: 99, current: 1 })
            .catch(() => {
                throw new Error()
            })
        if (config('http.code.success') === success && data && data.length > 0) {
            formData.value.details = data.map((item) => {
                let conditions = item.conditions
                // Handle both JSON string and array formats
                if (typeof conditions === 'string') {
                    try {
                        conditions = JSON.parse(conditions)
                    } catch {
                        conditions = []
                    }
                }
                let destinations = item.destinations
                // Handle both JSON string and array formats
                if (typeof destinations === 'string') {
                    try {
                        destinations = JSON.parse(destinations)
                    } catch {
                        destinations = []
                    }
                }
                return {
                    _id: item.id,
                    order: item.order || 0,
                    relation_type: item.relation_type || item.relationType || 'AND',
                    conditions:
                        Array.isArray(conditions) && conditions.length > 0
                            ? conditions.map((c) => ({
                                  ...c,
                                  op_type: c.op_type !== undefined ? c.op_type : c.opType,
                              }))
                            : [createEmptyCondition()],
                    destinations:
                        Array.isArray(destinations) && destinations.length > 0
                            ? destinations.map((d) => ({
                                  ...d,
                                  relation_type: d.relation_type || d.relationType || 'AND',
                                  conditions:
                                      Array.isArray(d.conditions) && d.conditions.length > 0
                                          ? d.conditions.map((dc) => ({
                                                ...dc,
                                                op_type: dc.op_type !== undefined ? dc.op_type : dc.opType,
                                            }))
                                          : [createEmptyDestCondition()],
                              }))
                            : [createEmptyDestination()],
                }
            })
            activeDetailKeys.value = formData.value.details.map((_, i) => String(i))
        }
    } catch (error) {
        // ignore
    }
}

// ---- 保存 ----
async function saveRouteDetails(routeId) {
    // 获取已有的 detail IDs
    const existingIds = new Set()
    try {
        const { success, data } = await apis.policy
            .getRouteDetailList({ route_id: routeId, pageSize: 99, current: 1 })
            .catch(() => ({ success: false, data: [] }))
        if (config('http.code.success') === success && data) {
            data.forEach((item) => existingIds.add(item.id))
        }
    } catch {
        // ignore
    }

    const submittedIds = new Set()

    for (const detail of formData.value.details || []) {
        const params = {
            route_id: routeId,
            relation_type: detail.relation_type || 'AND',
            order: detail.order || 0,
            conditions: detail.conditions || undefined,
            destinations: detail.destinations || undefined,
            enabled: formData.value.enabled,
        }

        if (detail._id) {
            submittedIds.add(detail._id)
            await apis.policy.updateRouteDetail(detail._id, params).catch(() => {
                throw new Error()
            })
        } else {
            const result = await apis.policy.createRouteDetail(params).catch(() => {
                throw new Error()
            })
            if (result?.data?.id) {
                submittedIds.add(result.data.id)
            }
        }
    }

    // 删除不再存在的 detail
    for (const id of existingIds) {
        if (!submittedIds.has(id)) {
            await apis.policy.delRouteDetail(id).catch(() => {
                // ignore
            })
        }
    }
}

function handleCreate() {
    formData.value.enabled = 0
    formData.value.details = [createEmptyDetail()]
    activeDetailKeys.value = ['0']
    showModal({
        type: 'create',
        title: t('pages.tagRoute.add'),
    })
}

async function handleCopy(record = {}) {
    showModal({
        type: 'create',
        title: t('pages.tagRoute.copy'),
    })

    const { data, success } = await apis.policy.getRoute(record.id).catch()
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
    cloned.details = []
    formData.value = cloned
    await loadRouteDetails(record.id)
    if (!Array.isArray(formData.value.details) || formData.value.details.length === 0) {
        formData.value.details = [createEmptyDetail()]
        activeDetailKeys.value = ['0']
    }
}

async function handleEdit(record = {}) {
    showModal({
        type: 'edit',
        title: t('pages.tagRoute.edit'),
    })

    const { data, success } = await apis.policy.getRoute(record.id).catch()
    if (!success) {
        message.error(t('component.message.error.save'))
        hideModal()
        return
    }
    formRecord.value = data
    const cloned = cloneDeep(data)
    cloned.details = []
    formData.value = cloned
    await loadRouteDetails(record.id)
    if (!Array.isArray(formData.value.details) || formData.value.details.length === 0) {
        formData.value.details = [createEmptyDetail()]
        activeDetailKeys.value = ['0']
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
                    enabled: formData.value.enabled,
                    description: formData.value.description,
                }
                let result = null
                switch (modal.value.type) {
                    case 'create':
                        result = await apis.policy.createRoute(params).catch(() => {
                            throw new Error()
                        })
                        break
                    case 'edit':
                        result = await apis.policy.updateRoute(formData.value.id, params).catch(() => {
                            throw new Error()
                        })
                        break
                }
                if (config('http.code.success') === result?.success) {
                    const routeId = result.data?.id || formData.value.id
                    await saveRouteDetails(routeId)
                    hideLoading()
                    hideModal()
                    emit('ok')
                } else {
                    hideLoading()
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
    activeDetailKeys.value = []
}

defineExpose({
    handleCreate,
    handleEdit,
    handleCopy,
})
</script>

<style lang="less" scoped>
.details-section {
    margin-top: 16px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.rule-card {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 4px;
    margin-bottom: 16px;
}

.rule-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;

    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 4px 4px 0 0;
}

.rule-card-title {
    font-weight: 500;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.rule-remove-btn {
    font-size: 14px;
    color: #1890ff;
    cursor: pointer;

    &:hover {
        color: #40a9ff;
    }
}

.rule-card-body {
    padding: 16px;
}

.rule-field {
    display: flex;
    margin-bottom: 16px;

    &:last-child {
        margin-bottom: 0;
    }
}

.rule-field-label {
    width: 100px;
    flex-shrink: 0;

    font-size: 13px;
    font-weight: 500;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));

    &.required::before {
        display: inline-block;
        margin-right: 4px;
        color: #ff4d4f;
        font-size: 14px;
        font-family: SimSun, sans-serif;
        line-height: 1;
        content: '*';
    }
}

.rule-field-content {
    flex: 1;
    min-width: 0;
}

.relation-row {
    margin-bottom: 12px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));
}

.rule-table {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 4px;
}

.rule-table-header {
    display: flex;
    padding: 8px 12px;

    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    font-weight: 500;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));

    gap: 12px;
}

.rule-table-row {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    gap: 12px;

    &:last-child {
        border-bottom: none;
    }
}

.col-type {
    flex: 0 0 100px;
}
.col-key {
    flex: 0 0 120px;
}
.col-op {
    flex: 0 0 100px;
    display: flex;
    align-items: center;
}
.col-value {
    flex: 1;
}
.col-action {
    flex: 0 0 60px;
    text-align: center;
}

.icon-btn {
    font-size: 16px;
    opacity: 0.6;
    cursor: pointer;
    margin: 0 4px;

    &:hover {
        color: #1890ff;
    }
}

.dest-item {
    display: flex;
    align-items: flex-start;
    margin-bottom: 16px;

    &:last-child {
        margin-bottom: 0;
    }
}

.dest-box {
    flex: 1;
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 4px;
    padding: 12px;
}

.dest-conditions-table {
    margin-bottom: 12px;
}

.dest-condition-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;

    &:last-child {
        margin-bottom: 0;
    }
}

.dest-col-key {
    flex: 0 0 150px;
}
.dest-col-op {
    flex: 0 0 120px;
}
.dest-col-value {
    flex: 1;
}
.dest-col-action {
    flex: 0 0 60px;
    text-align: center;
}

.dest-weight-row {
    display: flex;
    align-items: center;
    margin-top: 12px;
    color: var(--ant-color-text, rgba(0, 0, 0, 0.88));

    .required::before {
        display: inline-block;
        margin-right: 4px;
        color: #ff4d4f;
        font-size: 14px;
        font-family: SimSun, sans-serif;
        line-height: 1;
        content: '*';
    }
}

.dest-outer-action {
    flex: 0 0 60px;
    text-align: center;
    margin-top: 12px;
}

.add-rule-btn {
    padding-left: 0;
    margin-top: 8px;
}

.dark-form {
    .details-section {
        color: rgba(255, 255, 255, 0.85);
    }
    .rule-card {
        border-color: rgba(255, 255, 255, 0.15);
    }
    .rule-card-header {
        border-bottom-color: rgba(255, 255, 255, 0.15);
    }
    .rule-card-title {
        color: rgba(255, 255, 255, 0.85);
    }
    .rule-field-label {
        color: rgba(255, 255, 255, 0.85);
    }
    .relation-row {
        color: rgba(255, 255, 255, 0.85);
    }
    .rule-table {
        border-color: rgba(255, 255, 255, 0.15);
    }
    .rule-table-header {
        border-bottom-color: rgba(255, 255, 255, 0.15);
        color: rgba(255, 255, 255, 0.85);
    }
    .rule-table-row {
        border-bottom-color: rgba(255, 255, 255, 0.15);
    }
    .dest-box {
        border-color: rgba(255, 255, 255, 0.15);
    }
    .dest-weight-row {
        color: rgba(255, 255, 255, 0.85);
    }
}
</style>

<style lang="less">
html[data-theme='dark'] {
    .rule-card {
        background-color: #121214 !important;
        border-color: #27272a !important;

        .rule-card-header {
            background-color: #18181b !important;
            border-bottom-color: #27272a !important;
        }

        .rule-card-title {
            color: rgba(255, 255, 255, 0.85) !important;
        }

        .rule-field-label {
            color: rgba(255, 255, 255, 0.85) !important;
        }

        .relation-row {
            color: rgba(255, 255, 255, 0.85) !important;

            span {
                color: rgba(255, 255, 255, 0.85) !important;
            }
        }

        .rule-table {
            border-color: #27272a !important;

            .rule-table-header {
                background-color: #18181b !important;
                color: rgba(255, 255, 255, 0.85) !important;
                border-bottom-color: #27272a !important;

                div {
                    color: rgba(255, 255, 255, 0.85) !important;
                }
            }

            .rule-table-row {
                border-bottom-color: #27272a !important;
            }
        }

        .dest-box {
            background-color: #18181b !important;
            border-color: #27272a !important;
        }

        .dest-weight-row {
            color: rgba(255, 255, 255, 0.85) !important;

            span {
                color: rgba(255, 255, 255, 0.85) !important;
            }
        }
    }
}
</style>
