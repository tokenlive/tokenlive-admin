# Header Theme Toggle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add localized authenticated-header and login-page buttons that synchronize the overall, header, and side themes between light and dark.

**Architecture:** Add a shared `toggleTheme()` action to the Pinia app store, then call it from `BasicHeader.vue` and `UserLayout.vue`. The action updates the three theme fields and persists the shared config, so `ConfigDialog.vue` stays synchronized without additional state.

**Tech Stack:** Vue 3 Composition API, Pinia, Ant Design Vue, vue-i18n

---

### Task 1: Add The Synchronized Header Toggle

**Files:**
- Modify: `frontend/src/layouts/components/BasicHeader.vue`
- Modify: `frontend/src/locales/lang/en-US/settingDrawer.js`
- Modify: `frontend/src/locales/lang/zh-CN/settingDrawer.js`

- [ ] **Step 1: Run the static behavior check and verify it fails**

Run a Node script that requires a `handleThemeToggle` function, assignments to `theme`, `headerTheme`, and `sideTheme`, a call to `appStore.updateConfig()`, and matching locale keys.

- [ ] **Step 2: Add the theme toggle button**

Add the toggle before the settings button. Use `BulbOutlined` in light mode and `BulbFilled` in dark mode, wrapped in a localized tooltip.

- [ ] **Step 3: Add synchronized theme switching**

Compute the next theme from `config.theme`, assign it to all three theme fields, and call `appStore.updateConfig()`.

- [ ] **Step 4: Add Chinese and English tooltip translations**

Add `app.setting.theme.switch.light` and `app.setting.theme.switch.dark` to both setting drawer locale files.

- [ ] **Step 5: Verify**

Run Prettier, ESLint, the static behavior check, and `npm run build:prod`.
