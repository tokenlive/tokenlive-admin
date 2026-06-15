import { ApartmentOutlined, ProjectOutlined } from '@ant-design/icons-vue'

export default [
    {
        path: 'space',
        name: 'space',
        component: 'RouteViewLayout',
        meta: {
            icon: ApartmentOutlined,
            title: '空间管理',
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
        ],
    },
]
