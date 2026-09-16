import axios from 'axios'
import { config } from '@/config'
import { useUserStore } from '@/store'
import { createAPIKeyUsageLoader } from '@/utils/api-key-usage-request'

const transport = axios.create({ baseURL: config('http.apiBasic'), timeout: 10000 })

export function getAPIKeyRanking(params, options) {
    const user = useUserStore()
    const load = createAPIKeyUsageLoader({
        send: (request) => transport.request(request),
        getSession: () => ({ token: user.token, hasRefreshToken: user.hasRefreshToken }),
        refreshAccessToken: () => user.refreshAccessToken(),
        invalidateLocalSession: () => user.invalidateLocalSession(),
    })
    return load(params, options)
}
