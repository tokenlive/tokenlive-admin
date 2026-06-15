import {
    ControlOutlined,
    HighlightOutlined,
    DashboardOutlined,
    SafetyCertificateOutlined,
    BranchesOutlined,
    SlidersOutlined,
    ThunderboltOutlined,
} from '@ant-design/icons-vue'

export default [
    {
        path: 'policy',
        name: 'policy',
        component: 'RouteViewLayout',
        meta: {
            icon: ControlOutlined,
            title: '治理策略',
            isMenu: true,
            keepAlive: true,
            permission: '*',
        },
        children: [
            {
                path: 'tagging',
                name: 'taggingList',
                component: 'policy/tagging.vue',
                meta: {
                    icon: HighlightOutlined,
                    title: '流量染色策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'limit',
                name: 'limitList',
                component: 'policy/limit.vue',
                meta: {
                    icon: DashboardOutlined,
                    title: '限流策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'invocation',
                name: 'invocationList',
                component: 'policy/invocation.vue',
                meta: {
                    icon: SafetyCertificateOutlined,
                    title: '调用容错策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'route',
                name: 'routeList',
                component: 'policy/tagRoute.vue',
                meta: {
                    icon: BranchesOutlined,
                    title: '标签路由策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'loadbalance',
                name: 'loadbalanceList',
                component: 'policy/loadbalance.vue',
                meta: {
                    icon: SlidersOutlined,
                    title: '负载均衡策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
            {
                path: 'circuit-break',
                name: 'circuitBreakList',
                component: 'policy/circuitBreak.vue',
                meta: {
                    icon: ThunderboltOutlined,
                    title: '熔断降级策略',
                    isMenu: true,
                    keepAlive: true,
                    permission: '*',
                },
            },
        ],
    },
]
