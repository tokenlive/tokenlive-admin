import { message } from 'ant-design-vue'
import jschardet from 'jschardet'
import XYHttp from 'xy-http'
import { config } from '@/config'

import { useUserStore } from '@/store'

const MSG_ERROR_KEY = Symbol('GLOBAL_ERROR')

// 刷新 token 状态锁
let isRefreshing = false
// 请求队列
let requestQueue = []

/**
 * 将请求加入队列
 * @param {Function} callback
 */
const addRequestToQueue = (callback) => {
    requestQueue.push(callback)
}

/**
 * 执行队列中所有请求
 * @param {string} newToken
 */
const processQueue = (newToken) => {
    requestQueue.forEach((callback) => callback(newToken))
    requestQueue = []
}

/**
 * 清空队列
 */
const clearQueue = () => {
    requestQueue = []
}

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
                userStore.logout()
                return Promise.reject(err)
            }

            // 如果没有 refresh token，直接登出
            if (!userStore.hasRefreshToken) {
                userStore.logout()
                return Promise.reject(err)
            }

            // 如果正在刷新 token，将请求加入队列
            if (isRefreshing) {
                return new Promise((resolve) => {
                    addRequestToQueue((newToken) => {
                        originalRequest.headers['Authorization'] = newToken
                        resolve(new XYHttp(options).request(originalRequest))
                    })
                })
            }

            // 开始刷新 token
            isRefreshing = true

            try {
                const refreshSuccess = await userStore.refreshAccessToken()
                if (refreshSuccess) {
                    const newToken = userStore.token
                    originalRequest.headers['Authorization'] = newToken
                    processQueue(newToken)
                    return new XYHttp(options).request(originalRequest)
                } else {
                    clearQueue()
                    userStore.logout()
                    return Promise.reject(err)
                }
            } catch (refreshError) {
                clearQueue()
                userStore.logout()
                return Promise.reject(refreshError)
            } finally {
                isRefreshing = false
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
