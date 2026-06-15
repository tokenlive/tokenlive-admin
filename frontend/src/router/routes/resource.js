import { AppstoreOutlined, ClusterOutlined, DeploymentUnitOutlined } from '@ant-design/icons-vue'

export default [
    {
        path: 'resource',
        name: 'resource',
        component: 'RouteViewLayout',
        meta: {
            icon: AppstoreOutlined,
            title: '基础资源',
            isMenu: true,
            keepAlive: true,
            permission: '*',
        },
        children: [
            {
                path: 'provider',
                name: 'providerList',
                component: 'resource/provider.vue',
                meta: {
                    icon: ClusterOutlined,
                    title: '供应商管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'model',
                name: 'modelList',
                component: 'resource/model.vue',
                meta: {
                    icon: DeploymentUnitOutlined,
                    title: '模型管理',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'model/:id',
                name: 'modelDetail',
                component: 'resource/ModelDetail.vue',
                meta: {
                    title: '模型详情',
                    isMenu: false,
                    keepAlive: false,
                    permission: '*',
                    active: 'modelList',
                    openKeys: ['resource'],
                    breadcrumb: [
                        { name: 'resource', meta: { title: '基础资源' } },
                        { name: 'modelList', meta: { title: '模型管理' } },
                    ],
                },
            },
            {
                path: 'provider/:id',
                name: 'providerDetail',
                component: 'resource/ProviderDetail.vue',
                meta: {
                    title: '供应商详情',
                    isMenu: false,
                    keepAlive: false,
                    permission: '*',
                    active: 'providerList',
                    openKeys: ['resource'],
                    breadcrumb: [
                        { name: 'resource', meta: { title: '基础资源' } },
                        { name: 'providerList', meta: { title: '供应商管理' } },
                    ],
                },
            },
        ],
    },
]
