<template>
    <div
        class="basic-menu"
        ref="basicMenuRef">
        <a-menu
            v-model:selected-keys="selectedKeys"
            :get-pop-container="() => basicMenuRef"
            :inline-collapsed="collapsed"
            :mode="mode"
            :open-keys="cpOpenKeys"
            :theme="theme"
            :items="items"
            @openChange="onOpenChange"
            @click="handleClick"></a-menu>
    </div>
</template>

<script setup>
import { computed, onMounted, ref, watch, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { mapping } from '@/utils/util'
import { Badge } from 'ant-design-vue'

defineOptions({
    name: 'BasicMenu',
})

/**
 * @property {string} theme 主题，【dark=暗色，light=亮色】
 * @property {string} mode 菜单类型，【vertical=垂直，horizontal=水平，inline=内嵌】
 * @property {array} dataList 数据
 */
const props = defineProps({
    theme: {
        type: String,
        default: 'dark',
    },
    mode: {
        type: String,
        default: 'inline',
    },
    dataList: {
        type: Array,
        default: () => [],
    },
    isGroup: {
        type: Boolean,
        default: true,
    },
})

const emit = defineEmits(['click'])

const route = useRoute()
const router = useRouter()

const collapsed = ref(false)
const openKeys = ref([])
const selectedKeys = ref([])
const items = ref([])
const basicMenuRef = ref()

const cpIsHorizontal = computed(() => props.mode === 'horizontal')
const cpOpenKeys = computed(() => {
    if (cpIsHorizontal.value) {
        return []
    }
    return openKeys.value
})

watch(route, () => setSelectedMenu())
watch(
    () => props.dataList,
    () => {
        const mappedItems = mapping({
            data: props.dataList || [],
            fieldNames: {
                key: 'name',
                label: (item) =>
                    h('span', { class: 'basic-menu__title' }, [
                        h('span', { class: 'basic-menu__name' }, item?.meta?.title),
                        h(Badge, { count: item?.meta?.badge || 0 }),
                    ]),
                icon: (item) => {
                    const icon = item?.meta?.icon
                    if (icon) {
                        return h(icon)
                    }
                    return ''
                },
                children: 'children',
            },
            treeFieldName: 'children',
            keepOtherFields: true,
        })

        if (props.isGroup) {
            items.value = mappedItems.map((item) => {
                if (item.children && item.children.length > 0) {
                    return {
                        ...item,
                        type: 'group',
                        label: h(
                            'span',
                            { class: 'basic-menu__title', style: { display: 'flex', alignItems: 'center' } },
                            [
                                h('span', { class: 'basic-menu__name' }, item.meta?.title || item.name),
                                h(Badge, { count: item.meta?.badge || 0 }),
                            ]
                        ),
                    }
                }
                return item
            })
        } else {
            items.value = mappedItems
        }
    },
    { immediate: true, deep: true }
)

onMounted(() => {
    setSelectedMenu()
})

/**
 * 设置选中菜单
 */
function setSelectedMenu() {
    const { meta } = route || {}
    const keys = meta?.openKeys || []
    openKeys.value = Array.from(new Set([...openKeys.value, ...keys]))
    selectedKeys.value = meta?.breadcrumb.map((item) => item?.meta?.active || item.name)
}

/**
 * 点击菜单
 * @param item
 */
function handleClick({ item }) {
    const { path, meta, name, props } = item?.originItemValue || {}

    if (props) {
        props?.click?.call(null, item?.originItemValue)
    }

    if (path) {
        const isBlank = meta?.target === '_blank'
        const { href } = router.resolve({ name, query: meta?.query || {} })
        if (meta?.isLink) {
            if (isBlank) {
                window.open(href)
            } else {
                window.location.href = href
            }
        } else {
            if (isBlank) {
                window.open(href)
            } else {
                router.push({
                    path,
                    query: meta?.query ?? {},
                })
            }
        }
    }

    emit('click', item?.originItemValue)
}

/**
 * SubMenu 展开/关闭的回调
 * @param value
 */
function onOpenChange(value) {
    if (cpIsHorizontal.value) return
    openKeys.value = value
}
</script>

<style lang="less" scoped>
.basic-menu {
    .ant-menu:not(.ant-menu-horizontal) {
        :deep(.ant-menu-submenu-title) {
            display: flex;
        }

        :deep(.basic-menu) {
            &__title {
                flex: 1;
                display: flex;
                align-items: center;
                min-width: 0;
                overflow: hidden;
                text-overflow: ellipsis;
            }
            &__name {
                flex: 1;
                min-width: 0;
                overflow: hidden;
                text-overflow: ellipsis;
            }
        }

        :deep(.ant-menu-item-group-title) {
            font-size: 12px;
            .basic-menu__name {
                font-size: 12px;
            }
        }

        &.ant-menu-dark {
            :deep(.ant-menu-item-group-title) {
                color: rgba(255, 255, 255, 0.45);
            }
        }

        &.ant-menu-light {
            :deep(.ant-menu-item-group-title) {
                color: rgba(0, 0, 0, 0.45);
            }
        }
    }

    :deep(.ant-badge) {
        zoom: 0.8;
        margin: 0 1px 0 2px;
    }
}
</style>
