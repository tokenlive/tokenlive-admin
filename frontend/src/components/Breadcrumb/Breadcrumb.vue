<template>
    <a-breadcrumb
        class="x-breadcrumb"
        :class="{ 'x-breadcrumb--dark': theme === 'dark' }">
        <a-breadcrumb-item
            v-for="item in breadcrumbList"
            :key="item.name">
            {{ item.title }}
        </a-breadcrumb-item>
    </a-breadcrumb>
</template>

<script setup>
import { ref, computed } from 'vue'
import { onBeforeRouteUpdate, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

defineOptions({
    name: 'XBreadcrumb',
})

defineProps({
    theme: {
        type: String,
        default: 'light',
    },
})

const route = useRoute()
const { t } = useI18n()

const breadcrumbData = ref([])

const breadcrumbList = computed(() =>
    (breadcrumbData.value || []).map((item) => ({
        ...item,
        title: t(item.name, item.meta?.title || ''),
    }))
)

update()

onBeforeRouteUpdate((to) => {
    update(to)
})

function update(_route = route) {
    breadcrumbData.value = _route?.meta?.breadcrumb
}
</script>

<style lang="less" scoped>
.x-breadcrumb {
    display: flex;
    align-items: center;
    padding: 0 12px;

    &--dark {
        :deep(.ant-breadcrumb-separator),
        :deep(.ant-breadcrumb-link) {
            color: rgba(255, 255, 255, 0.65);
        }

        :deep(.ant-breadcrumb > span:last-child .ant-breadcrumb-link) {
            color: rgba(255, 255, 255, 0.85);
        }
    }
}
</style>
