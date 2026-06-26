import { defineStore } from 'pinia'
import { config } from '@/config'
import storage from '@/utils/storage'
import apis from '@/apis'

import useAppStore from './app'
import useMultiTab from './multiTab'
import useRouter from './router'

const useUserStore = defineStore('user', {
    state: () => ({
        userInfo: storage.local.getItem(config('storage.userInfo'), null),
        token: storage.local.getItem(config('storage.token'), ''),
        refreshToken: storage.local.getItem(config('storage.refreshToken'), ''),
        refreshExpiresAt: storage.local.getItem(config('storage.refreshExpiresAt'), null),
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
        login(params) {
            return new Promise((resolve, reject) => {
                ;(async () => {
                    try {
                        const result = await apis.user.login(params).catch(() => {
                            throw new Error()
                        })
                        const { success, data } = result || {}
                        if (config('http.code.success') === success) {
                            const { access_token, refresh_token, expires_at } = data
                            this.token = access_token
                            storage.local.setItem(config('storage.token'), access_token)

                            // 只有当 remember_me 为 true 时才会返回 refresh_token
                            if (refresh_token) {
                                this.refreshToken = refresh_token
                                this.refreshExpiresAt = expires_at
                                storage.local.setItem(config('storage.refreshToken'), refresh_token)
                                storage.local.setItem(config('storage.refreshExpiresAt'), expires_at)
                            } else {
                                // 没有 remember_me，清除旧的 refresh token
                                this.clearRefreshToken()
                            }

                            await this.getUserInfo()
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
                const result = await apis.user.refreshToken({ refreshToken: this.refreshToken })
                const { success, data } = result || {}
                if (config('http.code.success') === success) {
                    const { access_token, refresh_token, expires_at } = data
                    this.token = access_token
                    storage.local.setItem(config('storage.token'), access_token)

                    // 滑动过期：更新 refresh token
                    if (refresh_token) {
                        this.refreshToken = refresh_token
                        this.refreshExpiresAt = expires_at
                        storage.local.setItem(config('storage.refreshToken'), refresh_token)
                        storage.local.setItem(config('storage.refreshExpiresAt'), expires_at)
                    }

                    return true
                }
                return false
            } catch (error) {
                // refresh token 失效，清除所有 token
                this.clearTokens()
                return false
            }
        },
        /**
         * 清除 refresh token
         */
        clearRefreshToken() {
            this.refreshToken = ''
            this.refreshExpiresAt = null
            storage.local.removeItem(config('storage.refreshToken'))
            storage.local.removeItem(config('storage.refreshExpiresAt'))
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
         * 退出登录
         */
        logout() {
            return new Promise((resolve) => {
                const appStore = useAppStore()
                const multiTab = useMultiTab()
                const router = useRouter()

                // 清除所有 token
                this.clearTokens()
                storage.local.removeItem(config('storage.userInfo'))
                this.$reset()
                appStore.$reset()
                multiTab.$reset()
                router.$reset()
                resolve()
            })
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
