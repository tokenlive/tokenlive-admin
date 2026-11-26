<template>
    <a-auto-complete
        :value="props.value ?? undefined"
        :placeholder="props.placeholder"
        :options="displayedOptions"
        :filter-option="false"
        :default-active-first-option="false"
        :show-arrow="true"
        :allow-clear="props.allowClear"
        :disabled="props.disabled"
        style="width: 100%"
        @update:value="handleUpdateValue"
        @change="handleChange"
        @select="handleSelect"
        @search="handleSearch"
        @blur="handleBlur"
        @focus="handleFocus"
        @dropdown-visible-change="handleDropdownVisibleChange"
        @mousedown="handleMousedown" />
</template>

<script setup>
import { computed, ref } from 'vue'
import { CONTEXT_LENGTH_OPTIONS, getContextLengthDropdownOptions } from '@/enums/model'

const props = defineProps({
    value: {
        default: undefined,
    },
    placeholder: {
        type: String,
        default: '',
    },
    options: {
        type: Array,
        default: () => CONTEXT_LENGTH_OPTIONS,
    },
    allowClear: {
        type: Boolean,
        default: true,
    },
    disabled: {
        type: Boolean,
        default: undefined,
    },
})

const emit = defineEmits(['update:value', 'change', 'select', 'search', 'blur', 'focus', 'dropdownVisibleChange'])

const isFiltering = ref(false)
const searchKeyword = ref('')

const displayedOptions = computed(() =>
    getContextLengthDropdownOptions(props.options, searchKeyword.value, isFiltering.value)
)

function resetFilter() {
    isFiltering.value = false
    searchKeyword.value = ''
}

function handleUpdateValue(val) {
    emit('update:value', val)
}

function handleChange(val, option) {
    emit('change', val, option)
}

function handleSearch(val) {
    const query = String(val ?? '').trim()
    searchKeyword.value = query
    isFiltering.value = query.length > 0
    emit('search', val)
}

function handleSelect(val, option) {
    resetFilter()
    emit('select', val, option)
}

function handleBlur(event) {
    resetFilter()
    emit('blur', event)
}

function handleFocus(event) {
    emit('focus', event)
}

function handleDropdownVisibleChange(open) {
    if (!open) {
        resetFilter()
    }
    emit('dropdownVisibleChange', open)
}

function handleMousedown() {
    if (isFiltering.value && displayedOptions.value.length === 0) {
        resetFilter()
    }
}
</script>
