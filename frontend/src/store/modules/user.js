import { defineStore } from 'pinia'
import { config } from '@/config'
import storage from '@/utils/storage'
import apis from '@/apis'
import { createRefreshCoordinator, getRefreshFailureAction } from '@/utils/session'

import useAppStore from './app'
import useMultiTab from './multiTab'
import useRouter from './router'

const refreshCoordinator = createRefreshCoordinator()

const useUserStore = defineStore('user', {
    state: () => ({
        userInfo: storage.local.getItem(config('storage.userInfo'), null),
        token: storage.local.getItem(config('storage.token'), ''),
        refreshToken: storage.local.getItem(config('storage.refreshToken'), ''),
        permission: storage.local.getItem(config('storage.permission'), []),
    }),
    getters: {
        isLogin: (state) => !!state.token,
        hasRefreshToken: (state) => !!state.refreshToken,
    },
    actions: {
        /**
         * 登录
         * @param {object} params
         * @returns {Promise<unknown>}
         */
        async applyLoginToken(data) {
            const { access_token, refresh_token } = data
            this.token = access_token
            storage.local.setItem(config('storage.token'), access_token)

            if (refresh_token) {
                this.refreshToken = refresh_token
                storage.local.setItem(config('storage.refreshToken'), refresh_token)
            } else {
                this.clearRefreshToken()
            }

            await this.getUserInfo()
        },
        login(params) {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.user.login(params).catch(() => {
                            throw new Error()
                        })
                        const { success, data } = result || {}
                        if (config('http.code.success') === success) {
                            await this.applyLoginToken(data)
                        }
                        resolve(result)
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
        oauthExchange(ticket) {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.user.oauthExchange({ ticket }).catch(() => {
                            throw new Error()
                        })
                        const { success, data } = result || {}
                        if (config('http.code.success') === success) {
                            await this.applyLoginToken(data)
                        }
                        resolve(result)
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
        /**
         * 用 refresh token 刷新 access token
         * @returns {Promise<boolean>}
         */
        async refreshAccessToken() {
            if (!this.refreshToken) {
                return false
            }

            try {
                const result = await refreshCoordinator.run(() =>
                    apis.user.refreshToken({ refresh_token: this.refreshToken })
                )
                const { success, data } = result || {}
                if (config('http.code.success') === success) {
                    const { access_token, refresh_token } = data
                    this.token = access_token
                    storage.local.setItem(config('storage.token'), access_token)

                    // 滑动过期：更新 refresh token
                    if (refresh_token) {
                        this.refreshToken = refresh_token
                        storage.local.setItem(config('storage.refreshToken'), refresh_token)
                    }

                    return true
                }
                return false
            } catch (error) {
                if (getRefreshFailureAction(error) === 'invalidate') {
                    this.invalidateLocalSession()
                    return false
                }
                throw error
            }
        },
        /**
         * 清除 refresh token
         */
        clearRefreshToken() {
            this.refreshToken = ''
            storage.local.removeItem(config('storage.refreshToken'))
        },
        /**
         * 清除所有 token
         */
        clearTokens() {
            this.token = ''
            storage.local.removeItem(config('storage.token'))
            this.clearRefreshToken()
        },
        /**
         * 仅使本地会话失效，不调用后端接口
         */
        invalidateLocalSession() {
            const appStore = useAppStore()
            const multiTab = useMultiTab()
            const router = useRouter()

            this.clearTokens()
            storage.local.removeItem(config('storage.userInfo'))
            this.$reset()
            appStore.$reset()
            multiTab.$reset()
            router.$reset()
        },
        /**
         * 退出登录（仅撤销当前设备 token，不影响其他设备）
         */
        async logout() {
            const accessToken = this.token
            const refreshToken = this.refreshToken

            try {
                if (accessToken) {
                    await apis.user.logout({ refresh_token: refreshToken || undefined })
                }
            } catch (e) {
                // 主动退出时忽略撤销接口错误，本地会话仍必须清理
            } finally {
                this.invalidateLocalSession()
            }
        },
        /**
         * 获取用户详情
         */
        getUserInfo() {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.user.getUserDetail().catch(() => {
                            throw new Error()
                        })
                        const { success, data } = result || {}
                        if (config('http.code.success') === success) {
                            this.userInfo = data
                            storage.local.setItem(config('storage.userInfo'), this.userInfo)
                            resolve(result)
                        } else {
                            throw new Error()
                        }
                    } catch (error) {
                        reject()
                    }
                })()
            })
        },
    },
})

export default useUserStore
