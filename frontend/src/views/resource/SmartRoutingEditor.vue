<template>
    <section class="smart-routing-editor">
        <a-alert
            type="info"
            show-icon
            :message="$t('pages.model.smart.capabilities')"
            :description="$t('pages.model.smart.references_hint')" />
        <a-alert
            v-if="!model.smart_routing_ready"
            type="warning"
            show-icon
            :message="$t('pages.model.smart.not_configured')"
            :description="$t('pages.model.smart.enable_after_config')" />
        <div class="smart-routing-heading">
            <span>{{ $t('pages.model.smart.ranges') }}</span>
            <a-space>
                <template v-if="editing">
                    <a-button
                        :disabled="saving"
                        @click="cancelEditing"
                        >{{ $t('button.cancel') }}</a-button
                    >
                    <a-button
                        type="primary"
                        :loading="saving"
                        :disabled="candidatesLoading || candidatesError"
                        @click="save"
                        >{{ $t('button.save') }}</a-button
                    >
                </template>
                <a-button
                    v-else
                    type="primary"
                    @click="startEditing">
                    {{ $t(model.smart_routing ? 'common.edit' : 'pages.model.smart.configure') }}
                </a-button>
            </a-space>
        </div>
        <a-alert
            v-if="candidatesError"
            type="error"
            show-icon
            :message="$t('pages.model.smart.candidates_load')">
            <template #action>
                <a-button
                    size="small"
                    @click="loadCandidates"
                    >{{ $t('pages.model.smart.retry') }}</a-button
                >
            </template>
        </a-alert>
        <a-alert
            v-else-if="editing && !candidatesLoading && candidateModels.length < 2"
            type="warning"
            show-icon
            :message="$t('pages.model.smart.not_enough_models')" />
        <template v-if="editing">
            <a-form
                :model="draft"
                :disabled="saving"
                layout="vertical">
                <a-form-item :label="$t('pages.model.smart.judge')">
                    <a-select
                        v-model:value="draft.judge_model_id"
                        show-search
                        option-filter-prop="label"
                        :loading="candidatesLoading"
                        :options="candidateOptions"
                        :placeholder="$t('pages.model.smart.select_model')" />
                </a-form-item>
                <div class="smart-judge-limits">
                    <a-form-item :label="$t('pages.model.smart.timeout')">
                        <a-input-number
                            v-model:value="draft.judge_timeout_ms"
                            :min="1"
                            :precision="0"
                            style="width: 100%" />
                    </a-form-item>
                    <a-form-item :label="$t('pages.model.smart.input_bytes')">
                        <a-input-number
                            v-model:value="draft.judge_max_input_bytes"
                            :min="1"
                            :precision="0"
                            style="width: 100%" />
                    </a-form-item>
                    <a-form-item :label="$t('pages.model.smart.output_tokens')">
                        <a-input-number
                            v-model:value="draft.judge_max_output_tokens"
                            :min="1"
                            :precision="0"
                            style="width: 100%" />
                    </a-form-item>
                </div>
                <p class="smart-routing-hint">{{ $t('pages.model.smart.ranges_hint') }}</p>
                <div class="smart-routing-table-wrap">
                    <table class="smart-routing-table">
                        <thead>
                            <tr>
                                <th>{{ $t('pages.model.smart.range') }}</th>
                                <th>{{ $t('pages.model.smart.split_point') }}</th>
                                <th>{{ $t('pages.model.smart.target') }}</th>
                                <th>{{ $t('button.action') }}</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr
                                v-for="(range, index) in draft.ranges"
                                :key="index">
                                <td>{{ formatRoutingRange(range, index === draft.ranges.length - 1) }}</td>
                                <td>
                                    <a-input-number
                                        v-if="index < draft.ranges.length - 1"
                                        :value="range.max"
                                        :aria-label="$t('pages.model.smart.split_point')"
                                        :min="range.min + 1"
                                        :max="draft.ranges[index + 1].max - 1"
                                        :precision="0"
                                        style="width: 90px"
                                        @change="changeSplitPoint(index, $event)" />
                                    <span v-else>100</span>
                                </td>
                                <td>
                                    <a-select
                                        v-model:value="range.model_id"
                                        show-search
                                        option-filter-prop="label"
                                        :aria-label="$t('pages.model.smart.target')"
                                        :loading="candidatesLoading"
                                        :options="candidateOptions"
                                        :placeholder="$t('pages.model.smart.select_model')"
                                        style="width: 100%; min-width: 200px" />
                                </td>
                                <td>
                                    <a-space>
                                        <a-button
                                            type="link"
                                            size="small"
                                            :disabled="range.max - range.min < 2"
                                            @click="addSplitPoint(index)">
                                            {{ $t('pages.model.smart.split') }}
                                        </a-button>
                                        <a-button
                                            v-if="index < draft.ranges.length - 1"
                                            type="link"
                                            size="small"
                                            :disabled="draft.ranges.length <= 2"
                                            @click="removeSplitPoint(index)">
                                            {{ $t('pages.model.smart.merge') }}
                                        </a-button>
                                    </a-space>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </a-form>
            <a-alert
                v-if="validationError"
                type="error"
                show-icon
                :message="$t(`pages.model.smart.validation.${validationError}`)" />
        </template>
        <template v-else-if="model.smart_routing">
            <a-descriptions
                bordered
                size="small"
                :column="2">
                <a-descriptions-item :label="$t('pages.model.smart.judge')">{{
                    referenceLabel(model.smart_routing.judge_model_id)
                }}</a-descriptions-item>
                <a-descriptions-item :label="$t('pages.model.smart.timeout')">{{
                    model.smart_routing.judge_timeout_ms
                }}</a-descriptions-item>
                <a-descriptions-item :label="$t('pages.model.smart.input_bytes')">{{
                    model.smart_routing.judge_max_input_bytes
                }}</a-descriptions-item>
                <a-descriptions-item :label="$t('pages.model.smart.output_tokens')">{{
                    model.smart_routing.judge_max_output_tokens
                }}</a-descriptions-item>
                <a-descriptions-item :label="$t('pages.model.smart.config_version')">{{
                    model.smart_routing.version
                }}</a-descriptions-item>
            </a-descriptions>
            <p class="smart-routing-hint">{{ $t('pages.model.smart.ranges_hint') }}</p>
            <a-table
                size="small"
                :pagination="false"
                :data-source="model.smart_routing.ranges"
                :columns="columns"
                :row-key="(range) => range.min">
                <template #bodyCell="{ column, record, index }">
                    <template v-if="column.key === 'range'">{{
                        formatRoutingRange(record, index === model.smart_routing.ranges.length - 1)
                    }}</template>
                    <template v-else-if="column.key === 'model_id'">{{ referenceLabel(record.model_id) }}</template>
                </template>
            </a-table>
        </template>
        <a-empty
            v-else
            :description="$t('pages.model.smart.empty_hint')" />
        <a-alert
            v-if="model.smart_routing"
            type="warning"
            show-icon
            :message="$t('pages.model.smart.dependencies_hint')" />
    </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { useI18n } from 'vue-i18n'
import apis from '@/apis'
import {
    normalizeSmartRouting,
    loadSmartModelCandidates,
    validateSmartRouting,
    updateSplitPoint,
    splitRoutingRange,
    removeRoutingSplit,
    formatRoutingRange,
    getModelSaveFeedback,
} from '@/utils/smart-model'

const props = defineProps({ model: { type: Object, required: true } })
const emit = defineEmits(['saved'])
const { t } = useI18n()
const editing = ref(false)
const saving = ref(false)
const draft = ref(normalizeSmartRouting())
const validationAttempted = ref(false)
const candidateModels = ref([])
const candidatesLoading = ref(false)
const candidatesError = ref(false)
const candidateOptions = computed(() =>
    candidateModels.value.map((model) => ({
        value: model.id,
        label: `${model.model_name} (${model.model_code})`,
    }))
)
const columns = computed(() => [
    { title: t('pages.model.smart.range'), key: 'range', width: 180 },
    { title: t('pages.model.smart.target'), key: 'model_id' },
])
const validationError = computed(() =>
    validationAttempted.value ? validateSmartRouting(draft.value, candidateModels.value) : null
)
let candidateRequest = 0
let saveRequest = 0

watch(
    () => [props.model.id, props.model.space_code, props.model.smart_routing],
    () => {
        editing.value = false
        validationAttempted.value = false
        saveRequest++
        saving.value = false
        loadCandidates()
    },
    { immediate: true }
)
onBeforeUnmount(() => {
    candidateRequest++
    saveRequest++
})

async function loadCandidates() {
    const request = ++candidateRequest
    candidateModels.value = []
    candidatesError.value = false
    candidatesLoading.value = true
    try {
        const models = await loadSmartModelCandidates(apis.model.getModelList, {
            spaceCode: props.model.space_code,
            modelId: props.model.id,
        })
        if (request === candidateRequest) candidateModels.value = models
    } catch {
        if (request === candidateRequest) candidatesError.value = true
    } finally {
        if (request === candidateRequest) candidatesLoading.value = false
    }
}

function startEditing() {
    draft.value = normalizeSmartRouting(props.model.smart_routing)
    validationAttempted.value = false
    editing.value = true
}
function cancelEditing() {
    if (saving.value) return
    editing.value = false
    validationAttempted.value = false
    draft.value = normalizeSmartRouting(props.model.smart_routing)
}
function referenceLabel(id) {
    const model = candidateModels.value.find((item) => item.id === id)
    return model ? `${model.model_name} (${model.model_code})` : id || '--'
}
function changeSplitPoint(index, value) {
    try {
        draft.value.ranges = updateSplitPoint(draft.value.ranges, index, value)
    } catch {
        message.warning(t('pages.model.smart.validation.partition'))
    }
}
function addSplitPoint(index) {
    const range = draft.value.ranges[index]
    draft.value.ranges = splitRoutingRange(draft.value.ranges, index, Math.floor((range.min + range.max) / 2))
}
function removeSplitPoint(index) {
    draft.value.ranges = removeRoutingSplit(draft.value.ranges, index)
}
async function save() {
    if (!editing.value || saving.value || candidatesLoading.value || candidatesError.value) return
    validationAttempted.value = true
    if (validationError.value) return
    const request = ++saveRequest
    saving.value = true
    try {
        const result = await apis.model.updateSmartRouting(props.model.id, normalizeSmartRouting(draft.value))
        if (request !== saveRequest) return
        if (!result?.success) throw new Error(result?.msg || t('component.message.error.save'))
        const feedback = getModelSaveFeedback(result.data)
        if (feedback) message.warning([t(`pages.model.smart.${feedback.key}`), ...feedback.warnings].join(' '))
        else message.success(t('pages.model.smart.routing_saved'))
        editing.value = false
        emit('saved')
    } catch (error) {
        if (request === saveRequest) {
            message.error(
                error?.response?.data?.error?.detail ||
                    error?.response?.data?.msg ||
                    error?.message ||
                    t('component.message.error.save')
            )
        }
    } finally {
        if (request === saveRequest) saving.value = false
    }
}
</script>

<style lang="less" scoped>
.smart-routing-editor {
    display: flex;
    flex-direction: column;
    gap: 16px;
}
.smart-routing-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}
.smart-judge-limits {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
}
.smart-routing-hint {
    color: var(--ant-color-text-secondary, #666);
    margin: 0;
}
.smart-routing-table-wrap {
    overflow-x: auto;
}
.smart-routing-table {
    width: 100%;
    border-collapse: collapse;
    th,
    td {
        padding: 12px 8px;
        text-align: left;
        border-bottom: 1px solid var(--ant-color-border-secondary, #f0f0f0);
    }
    th {
        font-weight: 500;
    }
}
@media (max-width: 768px) {
    .smart-judge-limits {
        grid-template-columns: 1fr;
        gap: 0;
    }
}
</style>
