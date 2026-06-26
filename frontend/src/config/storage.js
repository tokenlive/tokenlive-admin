import { env } from '@/utils/util'

export default {
    namespace: env('storageNamespace'),
    isLogin: 'is_login',
    token: 'token',
    refreshToken: 'refresh_token',
    refreshExpiresAt: 'refresh_expires_at',
    userInfo: 'user_info',
    permission: 'permission',
    config: 'config',
    lang: 'lang',
}
