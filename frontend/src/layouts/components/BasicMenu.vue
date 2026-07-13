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
            height: 32px;
            line-height: 32px;
            letter-spacing: 0.04em;
            padding: 5px 10px 0 35px;
            margin: 0;
        }

        &.ant-menu-dark {
            background: transparent;

            :deep(.ant-menu-item),
            :deep(.ant-menu-submenu-title) {
                height: 38px;
                line-height: 38px;
                margin: 3px 10px;
                border-radius: 8px;
                color: rgba(196, 207, 224, 0.68);
                transition:
                    background 0.2s ease,
                    color 0.2s ease,
                    transform 0.2s ease;
            }

            :deep(.ant-menu-item .anticon),
            :deep(.ant-menu-submenu-title .anticon) {
                color: rgba(196, 207, 224, 0.54);
            }

            :deep(.ant-menu-item:hover),
            :deep(.ant-menu-submenu-title:hover) {
                color: rgba(238, 244, 255, 0.92);
                background: rgba(148, 163, 184, 0.075);
            }

            :deep(.ant-menu-item-selected) {
                color: #eef4ff;
                background: linear-gradient(90deg, rgba(47, 140, 255, 0.2), rgba(47, 140, 255, 0.08));
                box-shadow: inset 0 0 0 1px rgba(47, 140, 255, 0.1);

                &::after {
                    display: none;
                }

                &::before {
                    content: '';
                    position: absolute;
                    top: 10px;
                    left: 0;
                    width: 2px;
                    height: 18px;
                    border-radius: 999px;
                    background: #2f8cff;
                    box-shadow: 0 0 10px rgba(47, 140, 255, 0.55);
                }
            }

            :deep(.ant-menu-item-selected .anticon) {
                color: #6eb6ff;
            }

            :deep(.ant-menu-item-group-title) {
                color: rgba(196, 207, 224, 0.42);
            }
        }

        &.ant-menu-light {
            background: transparent;

            :deep(.ant-menu-item),
            :deep(.ant-menu-submenu-title) {
                height: 38px;
                line-height: 38px;
                margin: 3px 10px;
                border-radius: 8px;
                color: rgba(31, 41, 55, 0.68);
                transition:
                    background 0.2s ease,
                    color 0.2s ease,
                    transform 0.2s ease;
            }

            :deep(.ant-menu-item .anticon),
            :deep(.ant-menu-submenu-title .anticon) {
                color: rgba(31, 41, 55, 0.48);
            }

            :deep(.ant-menu-item:hover),
            :deep(.ant-menu-submenu-title:hover) {
                color: rgba(17, 24, 39, 0.9);
                background: rgba(47, 140, 255, 0.055);
            }

            :deep(.ant-menu-item-selected) {
                color: #0f4fa8;
                background: linear-gradient(90deg, rgba(47, 140, 255, 0.13), rgba(47, 140, 255, 0.055));
                box-shadow: inset 0 0 0 1px rgba(47, 140, 255, 0.08);

                &::after {
                    display: none;
                }

                &::before {
                    content: '';
                    position: absolute;
                    top: 10px;
                    left: 0;
                    width: 2px;
                    height: 18px;
                    border-radius: 999px;
                    background: #2f8cff;
                    box-shadow: 0 0 10px rgba(47, 140, 255, 0.32);
                }
            }

            :deep(.ant-menu-item-selected .anticon) {
                color: #2f8cff;
            }

            :deep(.ant-menu-item-group-title) {
                color: rgba(31, 41, 55, 0.42);
            }
        }
    }

    :deep(.ant-badge) {
        zoom: 0.8;
        margin: 0 1px 0 2px;
    }
}
</style>
