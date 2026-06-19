import { LineChartOutlined, DesktopOutlined, HomeOutlined } from '@ant-design/icons-vue'

export default [
    {
        path: 'monitor',
        name: 'monitor',
        component: 'RouteViewLayout',
        meta: {
            icon: LineChartOutlined,
            title: '系统监控',
            isMenu: true,
            keepAlive: true,
            permission: '*',
        },
        children: [
            {
                path: 'home',
                name: 'home',
                component: 'home/index.vue',
                meta: {
                    icon: HomeOutlined,
                    title: '首页总览',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'ops',
                name: 'opsMonitor',
                component: 'ops/index.vue',
                meta: {
                    icon: DesktopOutlined,
                    title: '运维看板',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
        ],
    },
]
