<template>
    <div :class="['user-layout-container', { 'user-layout--dark': appStore.config.theme === 'dark' }]">
        <div class="user-layout-smoke">
            <canvas ref="smokeCanvasRef"></canvas>
            <div class="user-layout-smoke__veil"></div>
        </div>
        <div class="user-layout-orbit user-layout-orbit--north"></div>
        <div class="user-layout-orbit user-layout-orbit--south"></div>
        <div class="user-layout-grid"></div>

        <div class="user-layout-main">
            <div class="user-layout-content">
                <div class="user-layout-top">
                    <img
                        class="user-layout-logo"
                        alt=""
                        src="/images/logo.png" />
                    <div>
                        <div class="user-layout-kicker">{{ title }}</div>
                        <div class="user-layout-header">{{ $t('login') }}</div>
                    </div>
                </div>
                <div class="user-layout-desc">{{ $t('pages.layouts.userLayout.title') }}</div>
                <div class="user-layout-form">
                    <router-view></router-view>
                </div>
                <div class="user-layout-footer">© {{ title }} {{ version }}</div>
            </div>
        </div>

        <div class="basic-header__right">
            <a-space :size="16">
                <a-tooltip :title="themeToggleTitle">
                    <action-button @click="handleThemeToggle">
                        <template v-if="appStore.config.theme === 'dark'">
                            <svg
                                width="18"
                                height="18"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round">
                                <circle
                                    cx="12"
                                    cy="12"
                                    r="5"></circle>
                                <line
                                    x1="12"
                                    y1="1"
                                    x2="12"
                                    y2="3"></line>
                                <line
                                    x1="12"
                                    y1="21"
                                    x2="12"
                                    y2="23"></line>
                                <line
                                    x1="4.22"
                                    y1="4.22"
                                    x2="5.64"
                                    y2="5.64"></line>
                                <line
                                    x1="18.36"
                                    y1="18.36"
                                    x2="19.78"
                                    y2="19.78"></line>
                                <line
                                    x1="1"
                                    y1="12"
                                    x2="3"
                                    y2="12"></line>
                                <line
                                    x1="21"
                                    y1="12"
                                    x2="23"
                                    y2="12"></line>
                                <line
                                    x1="4.22"
                                    y1="19.78"
                                    x2="5.64"
                                    y2="18.36"></line>
                                <line
                                    x1="18.36"
                                    y1="5.64"
                                    x2="19.78"
                                    y2="4.22"></line>
                            </svg>
                        </template>
                        <template v-else>
                            <svg
                                width="18"
                                height="18"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round">
                                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
                            </svg>
                        </template>
                    </action-button>
                </a-tooltip>
                <a-dropdown :trigger="['hover']">
                    <action-button :style="{ height: '44px' }">
                        <translation-outlined />
                    </action-button>
                    <template #overlay>
                        <a-menu v-model:selectedKeys="current">
                            <a-menu-item
                                v-for="(item, key) in langData"
                                :key="key"
                                @click="handleLang(key)">
                                {{ item.icon }} {{ item.label }}
                            </a-menu-item>
                        </a-menu>
                    </template>
                </a-dropdown>
            </a-space>
        </div>
    </div>
</template>

<script setup>
import { config as conf, config } from '@/config'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { TranslationOutlined } from '@ant-design/icons-vue'
import { useAppStore } from '@/store'
import ActionButton from './components/ActionButton.vue'

import storage from '@/utils/storage'
import { useI18n } from 'vue-i18n'
const { locale, t } = useI18n()
const appStore = useAppStore()
defineOptions({
    name: 'UserLayout',
})

const { version } = __APP_INFO__
const title = config('app.title')
const defaultLang = storage.local.getItem(conf('storage.lang')) || 'zh-ch'
const current = ref(defaultLang)
const smokeCanvasRef = ref()
let smokeFrameId = 0
let smokeCleanup = null
const themeToggleTitle = computed(() =>
    t(appStore.config.theme === 'dark' ? 'app.setting.theme.switch.light' : 'app.setting.theme.switch.dark')
)
const langData = ref({
    'zh-ch': {
        lang: 'zh-ch',
        label: '简体中文',
        icon: '🇨🇳',
        title: '语言',
    },
    'en-us': {
        lang: 'en-us',
        label: 'English',
        icon: '🇺🇸',
        title: 'Language',
    },
})

/**
 * 切换语言
 */

function handleLang(lang) {
    storage.local.setItem(conf('storage.lang'), lang)
    locale.value = lang
    current.value = lang
    location.reload()
}

function handleThemeToggle() {
    appStore.toggleTheme()
}

const smokeVertexSource = `
attribute vec4 a_position;
void main() {
    gl_Position = a_position;
}
`

const smokeFragmentSource = `
precision mediump float;

uniform vec2 iResolution;
uniform float iTime;
uniform vec2 iMouse;
uniform vec3 uColor;

void mainImage(out vec4 fragColor, in vec2 fragCoord) {
    vec2 uv = fragCoord / iResolution;
    vec2 centeredUv = (2.0 * fragCoord - iResolution.xy) / min(iResolution.x, iResolution.y);
    float time = iTime * 0.45;
    vec2 mouse = iMouse / iResolution;
    vec2 rippleCenter = 2.0 * mouse - 1.0;
    vec2 distortion = centeredUv;

    for (float i = 1.0; i < 8.0; i++) {
        distortion.x += 0.5 / i * cos(i * 2.0 * distortion.y + time + rippleCenter.x * 3.1415);
        distortion.y += 0.5 / i * cos(i * 2.0 * distortion.x + time + rippleCenter.y * 3.1415);
    }

    float wave = abs(sin(distortion.x + distortion.y + time));
    float glow = smoothstep(0.9, 0.2, wave);
    float vignette = smoothstep(1.25, 0.1, length(centeredUv));
    vec3 color = mix(vec3(0.02, 0.06, 0.10), uColor, glow * vignette);
    color += vec3(0.0, 0.22, 0.18) * smoothstep(0.92, 0.18, abs(sin(distortion.x * 0.72 - distortion.y + time * 0.8))) * 0.32;
    fragColor = vec4(color, 1.0);
}

void main() {
    mainImage(gl_FragColor, gl_FragCoord.xy);
}
`

function initSmokeBackground() {
    const canvas = smokeCanvasRef.value
    if (!canvas) return

    const gl = canvas.getContext('webgl', { antialias: false, alpha: false })
    if (!gl) return

    const compileShader = (type, source) => {
        const shader = gl.createShader(type)
        gl.shaderSource(shader, source)
        gl.compileShader(shader)

        if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
            gl.deleteShader(shader)
            return null
        }

        return shader
    }

    const vertexShader = compileShader(gl.VERTEX_SHADER, smokeVertexSource)
    const fragmentShader = compileShader(gl.FRAGMENT_SHADER, smokeFragmentSource)
    if (!vertexShader || !fragmentShader) return

    const program = gl.createProgram()
    gl.attachShader(program, vertexShader)
    gl.attachShader(program, fragmentShader)
    gl.linkProgram(program)
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) return
    gl.useProgram(program)

    const positionBuffer = gl.createBuffer()
    gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer)
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]), gl.STATIC_DRAW)

    const positionLocation = gl.getAttribLocation(program, 'a_position')
    gl.enableVertexAttribArray(positionLocation)
    gl.vertexAttribPointer(positionLocation, 2, gl.FLOAT, false, 0, 0)

    const resolutionLocation = gl.getUniformLocation(program, 'iResolution')
    const timeLocation = gl.getUniformLocation(program, 'iTime')
    const mouseLocation = gl.getUniformLocation(program, 'iMouse')
    const colorLocation = gl.getUniformLocation(program, 'uColor')
    const mouse = { x: 0, y: 0, active: false }
    const startTime = Date.now()
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

    gl.uniform3f(colorLocation, 0.12, 0.34, 0.78)

    const handleMouseMove = (event) => {
        const rect = canvas.getBoundingClientRect()
        mouse.x = event.clientX - rect.left
        mouse.y = event.clientY - rect.top
    }
    const handleMouseEnter = () => {
        mouse.active = true
    }
    const handleMouseLeave = () => {
        mouse.active = false
    }

    const render = () => {
        const width = canvas.clientWidth
        const height = canvas.clientHeight
        const pixelRatio = Math.min(window.devicePixelRatio || 1, 2)
        const nextWidth = Math.floor(width * pixelRatio)
        const nextHeight = Math.floor(height * pixelRatio)

        if (canvas.width !== nextWidth || canvas.height !== nextHeight) {
            canvas.width = nextWidth
            canvas.height = nextHeight
        }

        gl.viewport(0, 0, canvas.width, canvas.height)
        gl.uniform2f(resolutionLocation, canvas.width, canvas.height)
        gl.uniform1f(timeLocation, reduceMotion ? 0 : (Date.now() - startTime) / 1000)
        gl.uniform2f(
            mouseLocation,
            mouse.active ? mouse.x * pixelRatio : canvas.width / 2,
            mouse.active ? (height - mouse.y) * pixelRatio : canvas.height / 2
        )
        gl.drawArrays(gl.TRIANGLES, 0, 6)

        if (!reduceMotion) {
            smokeFrameId = requestAnimationFrame(render)
        }
    }

    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('mouseenter', handleMouseEnter)
    canvas.addEventListener('mouseleave', handleMouseLeave)
    render()

    smokeCleanup = () => {
        cancelAnimationFrame(smokeFrameId)
        canvas.removeEventListener('mousemove', handleMouseMove)
        canvas.removeEventListener('mouseenter', handleMouseEnter)
        canvas.removeEventListener('mouseleave', handleMouseLeave)
        gl.deleteBuffer(positionBuffer)
        gl.deleteProgram(program)
        gl.deleteShader(vertexShader)
        gl.deleteShader(fragmentShader)
    }
}

onMounted(() => {
    initSmokeBackground()
})

onBeforeUnmount(() => {
    smokeCleanup?.()
})
</script>

<style lang="less" scoped>
.user-layout {
    &-container {
        min-height: 100vh;
        background: #07111f;
        color: #edf5ff;
        display: flex;
        overflow: hidden;
        position: relative;
        transition:
            background-color 0.3s ease,
            color 0.3s ease;

        &::before {
            content: '';
            position: absolute;
            inset: 0;
            background:
                linear-gradient(90deg, rgba(129, 196, 255, 0.08) 1px, transparent 1px),
                linear-gradient(180deg, rgba(129, 196, 255, 0.06) 1px, transparent 1px);
            background-size: 84px 84px;
            mask-image: radial-gradient(circle at center, #000 0%, transparent 72%);
            opacity: 0.52;
            z-index: 0;
        }
    }

    &-smoke {
        position: absolute;
        inset: 0;
        overflow: hidden;
        pointer-events: auto;
        z-index: 0;

        canvas {
            width: 100%;
            height: 100%;
            display: block;
        }

        &__veil {
            position: absolute;
            inset: 0;
            background:
                radial-gradient(
                    circle at 50% 42%,
                    transparent 0%,
                    rgba(7, 17, 31, 0.18) 34%,
                    rgba(7, 17, 31, 0.7) 100%
                ),
                linear-gradient(180deg, rgba(7, 17, 31, 0.16) 0%, rgba(7, 17, 31, 0.44) 100%);
            backdrop-filter: blur(4px);
            -webkit-backdrop-filter: blur(4px);
            pointer-events: none;
        }
    }

    &-main {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 72px 24px;
        position: relative;
        z-index: 1;
    }

    &-content {
        width: min(440px, 100%);
        min-height: 540px;
        padding: 34px 38px 28px;
        border: 1px solid rgba(255, 255, 255, 0.22);
        border-radius: 28px;
        background: rgba(255, 255, 255, 0.12);
        box-shadow:
            0 30px 90px rgba(0, 0, 0, 0.32),
            inset 0 1px 0 rgba(255, 255, 255, 0.22);
        backdrop-filter: blur(22px) saturate(150%);
        -webkit-backdrop-filter: blur(22px) saturate(150%);
    }

    &-header {
        display: flex;
        align-items: center;
        font-size: 34px;
        font-weight: 650;
        line-height: 1.08;
        color: #f5f9ff;
    }

    &-desc {
        margin: 14px 0 28px;
        color: rgba(237, 245, 255, 0.68);
        font-size: 14px;
        line-height: 1.7;
    }

    &-top {
        display: flex;
        align-items: center;
        gap: 16px;
    }

    &-logo {
        width: 62px;
        height: 62px;
        object-fit: contain;
        padding: 10px;
        border-radius: 20px;
        background: rgba(255, 255, 255, 0.12);
        box-shadow:
            0 14px 34px rgba(0, 0, 0, 0.22),
            inset 0 1px 0 rgba(255, 255, 255, 0.22);
    }

    &-kicker {
        margin-bottom: 6px;
        color: #68d5c4;
        font-size: 13px;
        font-weight: 600;
    }

    &-footer {
        margin-top: 22px;
        color: rgba(237, 245, 255, 0.56);
        font-size: 12px;
        text-align: center;
    }

    &-orbit {
        position: absolute;
        border-radius: 999px;
        filter: blur(2px);
        pointer-events: none;

        &--north {
            width: 460px;
            height: 460px;
            top: -160px;
            right: 9%;
            background: radial-gradient(circle, rgba(0, 121, 255, 0.16), transparent 66%);
        }

        &--south {
            width: 520px;
            height: 520px;
            left: 4%;
            bottom: -210px;
            background: radial-gradient(circle, rgba(0, 184, 148, 0.12), transparent 68%);
        }
    }

    &-grid {
        position: absolute;
        width: 760px;
        height: 760px;
        right: 50%;
        bottom: 50%;
        transform: translate(50%, 50%) rotate(-10deg);
        border: 1px solid rgba(129, 196, 255, 0.08);
        border-radius: 50%;
        pointer-events: none;
        z-index: 0;

        &::before,
        &::after {
            content: '';
            position: absolute;
            inset: 16%;
            border: 1px solid rgba(129, 196, 255, 0.08);
            border-radius: inherit;
        }

        &::after {
            inset: 31%;
        }
    }
}

.basic-header__right {
    color: #dbeafe;
    position: fixed;
    top: 18px;
    right: 18px;
    z-index: 2;
    padding: 10px 12px;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 999px;
    background: rgba(11, 23, 39, 0.46);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
}

.user-layout-container.user-layout--dark {
    background: #07111f;
    color: #edf5ff;

    .user-layout-content {
        border-color: rgba(255, 255, 255, 0.16);
        background: rgba(12, 23, 38, 0.58);
        box-shadow:
            0 30px 90px rgba(0, 0, 0, 0.32),
            inset 0 1px 0 rgba(255, 255, 255, 0.16);
    }

    .user-layout-kicker {
        color: #68d5c4;
    }

    .user-layout-desc,
    .user-layout-footer {
        color: rgba(237, 245, 255, 0.62);
    }

    .user-layout-logo {
        background: rgba(255, 255, 255, 0.1);
    }

    .user-layout-header {
        color: #f5f9ff;
    }

    .basic-header__right {
        color: #dbeafe;
        border-color: rgba(255, 255, 255, 0.12);
        background: rgba(11, 23, 39, 0.46);
    }

    .basic-header__right {
        :deep(.anticon) {
            color: var(--color-text-secondary) !important;
            &:hover {
                color: var(--color-text-primary) !important;
            }
        }
    }
}

@media (max-width: 640px) {
    .user-layout {
        &-main {
            align-items: flex-start;
            padding: 86px 16px 28px;
        }

        &-content {
            min-height: auto;
            padding: 28px 22px 24px;
            border-radius: 24px;
        }

        &-header {
            font-size: 28px;
        }

        &-logo {
            width: 54px;
            height: 54px;
            border-radius: 18px;
        }
    }

    .basic-header__right {
        top: 12px;
        right: 12px;
        padding: 8px 10px;
    }
}
</style>
