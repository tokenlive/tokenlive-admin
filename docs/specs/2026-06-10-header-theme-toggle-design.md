# Header Theme Toggle Design

## Goal

Add a light/dark theme toggle at the left edge of the authenticated header's right-side action area and beside the login page language button.

## Behavior

- The button shows a filled bulb in dark mode and an outlined bulb in light mode.
- Switching to dark sets `theme`, `headerTheme`, and `sideTheme` to `dark`.
- Switching to light sets `theme`, `headerTheme`, and `sideTheme` to `light`.
- The current `layout` and `menuMode` remain unchanged.
- The updated configuration is persisted through `appStore.updateConfig()`.
- Login and authenticated layouts call the same `appStore.toggleTheme()` action.
- The existing configuration drawer immediately reflects all three updated values because it binds to the same Pinia config object.
- The tooltip is localized in Chinese and English.

## Placement

The toggle is the first action in `BasicHeader`'s right-side action group and appears beside the language button in `UserLayout`.

## Verification

- A static behavior check verifies the button placement, all three assignments, persistence call, and locale keys.
- Run Prettier, ESLint, and the production frontend build.
