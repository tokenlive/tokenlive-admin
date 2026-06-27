import {
    ApartmentOutlined,
    ProjectOutlined,
    ClusterOutlined,
    DeploymentUnitOutlined,
    AppstoreOutlined,
    KeyOutlined,
} from '@ant-design/icons-vue'

export default [
    {
        path: 'space',
        name: 'space',
        component: 'RouteViewLayout',
        meta: {
            icon: ApartmentOutlined,
            title: '模型空间',
            isMenu: true,
            keepAlive: true,
            permission: '*',
        },
        children: [
            {
                path: 'list',
                name: 'spaceList',
                component: 'space/index.vue',
                meta: {
                    icon: ProjectOutlined,
                    title: '空间列表',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
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
                    openKeys: ['space'],
                    breadcrumb: [
                        { name: 'space', meta: { title: '模型空间' } },
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
                    openKeys: ['space'],
                    breadcrumb: [
                        { name: 'space', meta: { title: '模型空间' } },
                        { name: 'providerList', meta: { title: '供应商管理' } },
                    ],
                },
            },
            {
                path: 'model-catalog',
                name: 'modelCatalogList',
                component: 'resource/model_catalog.vue',
                meta: {
                    icon: AppstoreOutlined,
                    title: '模型目录',
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
                path: 'model-catalog/:id',
                name: 'modelCatalogDetail',
                component: 'resource/model_catalog_detail.vue',
                meta: {
                    title: '模型目录详情',
                    isMenu: false,
                    keepAlive: false,
                    permission: '*',
                    active: 'modelCatalogList',
                    openKeys: ['space'],
                    breadcrumb: [
                        { name: 'space', meta: { title: '模型空间' } },
                        { name: 'modelCatalogList', meta: { title: '模型目录' } },
                    ],
                },
            },
        ],
    },
]
