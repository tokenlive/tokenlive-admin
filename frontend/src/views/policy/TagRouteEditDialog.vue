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

            <!-- 路由规则列表 (RouteDetail 1:n) -->
            <a-form-item
                :label="$t('pages.tagRoute.form.details')"
                class="section-form-item">
                <template #label>
                    <span>
                        {{ $t('pages.tagRoute.form.details') }}
                        <a-tooltip :title="$t('pages.tagRoute.form.details.title')">
                            <question-circle-outlined style="margin-left: 4px; color: #999" />
                        </a-tooltip>
                    </span>
                </template>
                <div class="details-section">
                    <div
                        class="rule-card"
                        v-for="(detail, rIndex) in formData.details"
                        :key="String(rIndex)">
                        <div class="rule-card-header">
                            <span class="rule-card-title">
                                {{ $t('pages.tagRoute.form.details.rule') }} {{ rIndex + 1 }}
                            </span>
                            <a-button
                                v-if="formData.details.length > 1"
                                type="link"
                                size="small"
                                danger
                                @click="removeDetail(rIndex)">
                                <delete-outlined />
                            </a-button>
                        </div>
                        <div class="rule-card-body">
                            <!-- 匹配条件 -->
                            <div class="sub-section">
                                <div class="sub-section-title">
                                    {{ $t('pages.tagRoute.form.conditions') }}
                                </div>
                                <div class="condition-section">
                                    <div class="condition-row">
                                        <span class="condition-label">
                                            {{ $t('pages.tagRoute.form.relationType') }}
                                        </span>
                                        <a-radio-group v-model:value="detail.relation_type">
                                            <a-radio value="AND">
                                                AND({{ $t('pages.tagRoute.form.relationType.and') }})
                                            </a-radio>
                                            <a-radio value="OR">
                                                OR({{ $t('pages.tagRoute.form.relationType.or') }})
                                            </a-radio>
                                        </a-radio-group>
                                    </div>

                                    <div
                                        class="rule-table"
                                        v-if="detail.conditions && detail.conditions.length > 0">
                                        <div class="rule-table-header">
                                            <div class="col-type">
                                                {{ $t('pages.tagRoute.form.conditions.type') }}
                                            </div>
                                            <div class="col-key">
                                                {{ $t('pages.tagRoute.form.conditions.key') }}
                                            </div>
                                            <div class="col-op">
                                                {{ $t('pages.tagRoute.form.conditions.opType') }}
                                                <a-tooltip :title="$t('pages.tagRoute.form.conditions.opType.tooltip')">
                                                    <question-circle-outlined style="margin-left: 4px; color: #999" />
                                                </a-tooltip>
                                            </div>
                                            <div class="col-value">
                                                {{ $t('pages.tagRoute.form.conditions.values') }}
                                            </div>
                                            <div class="col-action">
                                                {{ $t('pages.tagRoute.form.conditions.action') }}
                                            </div>
                                        </div>
                                        <div
                                            class="rule-table-row"
                                            v-for="(condition, cIndex) in detail.conditions"
                                            :key="cIndex">
                                            <div class="col-type">
                                                <a-select
                                                    v-model:value="condition.type"
                                                    :placeholder="$t('pages.tagRoute.form.conditions.type.placeholder')"
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
                                                    style="width: 100%" />
                                            </div>
                                            <div class="col-op">
                                                <a-select
                                                    v-model:value="condition.op_type"
                                                    :placeholder="
                                                        $t('pages.tagRoute.form.conditions.opType.placeholder')
                                                    "
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
                                                    :placeholder="
                                                        $t('pages.tagRoute.form.conditions.values.placeholder')
                                                    "
                                                    style="width: 100%" />
                                            </div>
                                            <div class="col-action">
                                                <a-button
                                                    type="link"
                                                    size="small"
                                                    danger
                                                    @click="removeCondition(rIndex, cIndex)">
                                                    <delete-outlined />
                                                </a-button>
                                            </div>
                                        </div>
                                    </div>

                                    <a-button
                                        type="dashed"
                                        block
                                        size="small"
                                        @click="addCondition(rIndex)"
                                        style="margin-top: 8px">
                                        <plus-outlined />
                                        {{ $t('pages.tagRoute.form.conditions.add') }}
                                    </a-button>
                                </div>
                            </div>

                            <!-- 路由目标 -->
                            <div class="sub-section">
                                <div class="sub-section-title">
                                    {{ $t('pages.tagRoute.form.destinations') }}
                                    <a-tooltip :title="$t('pages.tagRoute.form.destinations.tooltip')">
                                        <question-circle-outlined style="margin-left: 4px; color: #999" />
                                    </a-tooltip>
                                </div>
                                <div class="condition-section">
                                    <div
                                        class="dest-item"
                                        v-for="(destination, dIndex) in detail.destinations"
                                        :key="dIndex">
                                        <div class="dest-box">
                                            <div
                                                class="rule-table"
                                                v-if="destination.conditions && destination.conditions.length > 0">
                                                <div class="rule-table-header">
                                                    <div class="dest-col-key">
                                                        {{ $t('pages.tagRoute.form.conditions.key') }}
                                                    </div>
                                                    <div class="dest-col-op">
                                                        {{ $t('pages.tagRoute.form.conditions.opType') }}
                                                    </div>
                                                    <div class="dest-col-value">
                                                        {{ $t('pages.tagRoute.form.conditions.values') }}
                                                    </div>
                                                    <div class="col-action">
                                                        {{ $t('pages.tagRoute.form.conditions.action') }}
                                                    </div>
                                                </div>
                                                <div
                                                    class="rule-table-row"
                                                    v-for="(dc, dcIndex) in destination.conditions"
                                                    :key="dcIndex">
                                                    <div class="dest-col-key">
                                                        <a-input
                                                            v-model:value="dc.key"
                                                            :placeholder="
                                                                $t('pages.tagRoute.form.conditions.key.placeholder')
                                                            "
                                                            style="width: 100%" />
                                                    </div>
                                                    <div class="dest-col-op">
                                                        <a-select
                                                            v-model:value="dc.op_type"
                                                            :placeholder="
                                                                $t('pages.tagRoute.form.conditions.opType.placeholder')
                                                            "
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
                                                            style="width: 100%" />
                                                    </div>
                                                    <div class="col-action">
                                                        <a-button
                                                            type="link"
                                                            size="small"
                                                            danger
                                                            @click="removeDestCondition(rIndex, dIndex, dcIndex)">
                                                            <delete-outlined />
                                                        </a-button>
                                                    </div>
                                                </div>
                                            </div>

                                            <a-button
                                                type="dashed"
                                                block
                                                size="small"
                                                @click="addDestCondition(rIndex, dIndex)"
                                                style="margin-top: 8px">
                                                <plus-outlined />
                                                {{ $t('pages.tagRoute.form.conditions.add') }}
                                            </a-button>

                                            <div class="condition-row dest-weight-row">
                                                <span class="condition-label required">
                                                    {{ $t('pages.tagRoute.form.destinations.weight') }}
                                                    <a-tooltip
                                                        :title="$t('pages.tagRoute.form.destinations.weight.tooltip')">
                                                        <question-circle-outlined
                                                            style="margin-left: 4px; color: #999" />
                                                    </a-tooltip>
                                                </span>
                                                <a-input-number
                                                    v-model:value="destination.weight"
                                                    :min="0"
                                                    style="width: 200px" />
                                            </div>
                                        </div>
                                        <div class="dest-action">
                                            <a-button
                                                v-if="detail.destinations.length > 1"
                                                type="link"
                                                size="small"
                                                danger
                                                @click="removeDestination(rIndex, dIndex)">
                                                <delete-outlined />
                                            </a-button>
                                        </div>
                                    </div>

                                    <a-button
                                        type="dashed"
                                        block
                                        size="small"
                                        @click="addDestination(rIndex)"
                                        style="margin-top: 8px">
                                        <plus-outlined />
                                        {{ $t('pages.tagRoute.form.destinations.add') }}
                                    </a-button>
                                </div>
                            </div>

                            <!-- 优先级 -->
                            <div class="sub-section">
                                <div class="sub-section-title">
                                    {{ $t('pages.tagRoute.form.order') }}
                                    <a-tooltip :title="$t('pages.tagRoute.form.order.tooltip')">
                                        <question-circle-outlined style="margin-left: 4px; color: #999" />
                                    </a-tooltip>
                                </div>
                                <div class="condition-section">
                                    <div class="condition-row">
                                        <span class="condition-label required">
                                            {{ $t('pages.tagRoute.form.order') }}
                                        </span>
                                        <a-input-number
                                            v-model:value="detail.order"
                                            :min="1"
                                            style="width: 200px" />
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <a-button
                        type="dashed"
                        block
                        @click="addDetail"
                        style="margin-top: 8px">
                        <plus-outlined />
                        {{ $t('pages.tagRoute.form.details.add') }}
                    </a-button>
                </div>
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
    formData.value.details.push(createEmptyDetail())
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
}

defineExpose({
    handleCreate,
    handleEdit,
    handleCopy,
})
</script>

<style lang="less" scoped>
.section-form-item {
    :deep(.ant-form-item-control-input-content) {
        overflow: visible;
    }
}

/* details-section 内文字颜色继承抽屉主体，暗色主题下自动适配 */

.rule-card {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    margin-bottom: 16px;

    &:last-of-type {
        margin-bottom: 0;
    }
}

.rule-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
}

.rule-card-title {
    font-size: 13px;
    font-weight: 500;
}

.rule-card-body {
    padding: 16px;
}

.sub-section {
    margin-bottom: 16px;

    &:last-child {
        margin-bottom: 0;
    }
}

.sub-section-title {
    font-size: 13px;
    font-weight: 500;
    margin-bottom: 8px;
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

.rule-table {
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    margin-bottom: 12px;
}

.rule-table-header {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    font-size: 13px;
    font-weight: 500;
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
    flex: 0 0 120px;
}
.col-key {
    flex: 0 0 140px;
}
.col-op {
    flex: 0 0 130px;
    display: flex;
    align-items: center;
}
.col-value {
    flex: 1;
    min-width: 0;
}
.col-action {
    flex: 0 0 56px;
    text-align: center;
}

.dest-item {
    display: flex;
    align-items: flex-start;
    margin-bottom: 12px;

    &:last-child {
        margin-bottom: 0;
    }
}

.dest-box {
    flex: 1;
    min-width: 0;
    border: 1px solid rgba(128, 128, 128, 0.2);
    border-radius: 6px;
    padding: 12px;
}

.dest-action {
    flex: 0 0 40px;
    text-align: center;
    margin-top: 4px;
}

.dest-col-key {
    flex: 0 0 160px;
}
.dest-col-op {
    flex: 0 0 140px;
}
.dest-col-value {
    flex: 1;
    min-width: 0;
}

.dest-weight-row {
    margin-top: 12px;
    margin-bottom: 0;
}
</style>
