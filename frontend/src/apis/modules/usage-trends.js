import axios from 'axios'
import { config } from '@/config'
import { useUserStore } from '@/store'
import { createAPIKeyUsageLoader } from '@/utils/api-key-usage-request'

const transport = axios.create({ baseURL: config('http.apiBasic'), timeout: 10000 })

export function getUsageTrends(params, options) {
    const user = useUserStore()
    return createAPIKeyUsageLoader({
        url: '/api/v1/dashboard/usage-trends',
        send: (request) => transport.request(request),
        getSession: () => ({ token: user.token, hasRefreshToken: user.hasRefreshToken }),
        refreshAccessToken: () => user.refreshAccessToken(),
        invalidateLocalSession: () => user.invalidateLocalSession(),
    })(params, options)
}
