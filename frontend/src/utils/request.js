import { message } from 'ant-design-vue'
import jschardet from 'jschardet'
import XYHttp from 'xy-http'
import { config } from '@/config'

import { useUserStore } from '@/store'

const MSG_ERROR_KEY = Symbol('GLOBAL_ERROR')

const options = {
    enableAbortController: true,
    interceptorRequest: (request) => {
        const userStore = useUserStore()
        const isLogin = userStore.isLogin
        const token = userStore.token

        if (isLogin) {
            request.headers['Authorization'] = token
        }
    },
    interceptorRequestCatch: () => {},
    interceptorResponse: (response) => {
        // 错误处理
        const { success, msg = 'Network Error' } = response.data || {}
        if (![true].includes(success)) {
            message.error({
                content: msg,
                key: MSG_ERROR_KEY,
            })
        }
    },
    interceptorResponseCatch: async (err) => {
        const userStore = useUserStore()
        const { success, error } = err.response?.data || {}
        const status = err.response?.status

        if (status === 401) {
            const originalRequest = err.config

            // 如果是 refresh-token 接口自己返回 401，说明 refresh token 失效了
            if (originalRequest.url.includes('/api/v1/refresh-token')) {
                return Promise.reject(err)
            }

            // 没有 refresh token 时只清理本地态，避免用失效 access token 递归调用 logout
            if (!userStore.hasRefreshToken) {
                userStore.invalidateLocalSession()
                return Promise.reject(err)
            }

            try {
                // user store 内部共享同一个 refresh Promise，并发 401 只会发起一次刷新
                const refreshSuccess = await userStore.refreshAccessToken()
                if (refreshSuccess) {
                    originalRequest.headers['Authorization'] = userStore.token
                    return new XYHttp(options).request(originalRequest)
                }
                return Promise.reject(err)
            } catch (refreshError) {
                return Promise.reject(refreshError)
            }
        }

        if ([false].includes(success)) {
            // Show error message to user
            message.error({
                content: error?.detail || 'Request failed',
                key: MSG_ERROR_KEY,
            })
        }
    },
}

/**
 * 读取文件
 */
class ReadFile extends XYHttp {
    constructor() {
        super({
            baseURL: '',
            responseType: 'blob',
            transformResponse: [
                async (data) => {
                    const encoding = await this._encoding(data)
                    return new Promise((resolve) => {
                        let reader = new FileReader()
                        reader.readAsText(data, encoding)
                        reader.onload = function () {
                            resolve(reader.result)
                        }
                    })
                },
            ],
        })
    }

    /**
     * 文本编码
     * @param data
     * @returns {Promise<unknown>}
     * @private
     */
    _encoding(data) {
        return new Promise((resolve) => {
            let reader = new FileReader()
            reader.readAsBinaryString(data)
            reader.onload = function () {
                resolve(jschardet.detect(reader?.result).encoding)
            }
        })
    }
}

const basic = new XYHttp({
    ...options,
    baseURL: config('http.apiBasic'),
})

const readFile = new ReadFile()

export default {
    basic,
    readFile,
}
