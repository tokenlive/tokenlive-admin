import {
    SettingOutlined,
    TeamOutlined,
    UsergroupAddOutlined,
    MenuOutlined,
    FileTextOutlined,
    KeyOutlined,
    ShopOutlined,
} from '@ant-design/icons-vue'

export default [
    {
        path: 'system',
        name: 'system',
        component: 'RouteViewLayout',
        meta: {
            icon: SettingOutlined,
            title: '系统管理',
            isMenu: true,
            keepAlive: true,
            permission: '*',
        },
        children: [
            {
                path: 'user',
                name: 'user',
                component: 'system/user/index.vue',
                meta: {
                    icon: TeamOutlined,
                    title: '成员与部门',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'role',
                name: 'role',
                component: 'system/role/index.vue',
                meta: {
                    icon: UsergroupAddOutlined,
                    title: '角色管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'menu',
                name: 'menu',
                component: 'system/menu/index.vue',
                meta: {
                    icon: MenuOutlined,
                    title: '菜单管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'logger',
                name: 'logger',
                component: 'system/logger/index.vue',
                meta: {
                    icon: FileTextOutlined,
                    title: '日志管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'user-api-key',
                name: 'user-api-key',
                component: 'system/user-api-key/index.vue',
                meta: {
                    icon: KeyOutlined,
                    title: 'API Key 管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'tenant',
                name: 'tenant',
                component: 'system/tenant/index.vue',
                meta: {
                    icon: ShopOutlined,
                    title: '租户管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'tenant/:id',
                name: 'tenantDetail',
                component: 'system/tenant/TenantDetail.vue',
                meta: {
                    title: '租户详情',
                    isMenu: false,
                    keepAlive: false,
                    permission: '*',
                    active: 'tenant',
                    openKeys: ['system'],
                    breadcrumb: [
                        { name: 'system', meta: { title: '系统管理' } },
                        { name: 'tenant', meta: { title: '租户管理' } },
                    ],
                },
            },
        ],
    },
]
