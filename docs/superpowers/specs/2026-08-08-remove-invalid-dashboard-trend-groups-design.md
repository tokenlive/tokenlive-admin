# Remove Invalid Dashboard Trend Groups

## Goal

Remove the `By Tenant` and `By Endpoint` choices from the Traffic and Success Rate chart on the Admin home page so users cannot select dimensions that do not currently provide meaningful data.

## Scope

- Remove the `tenant` and `endpoint` options from the home-page trend group selector.
- Remove the corresponding unused Chinese and English locale entries.
- Keep the dashboard trends API and its accepted `group_by` values unchanged for backward compatibility.
- Keep the existing Global, By Model, and By Provider choices unchanged.

## Implementation

The change is limited to the home-page Vue component and the two dashboard locale files. No backend, Gateway, Prometheus, Redis, or ClickHouse behavior changes.

## Verification

- Add a focused source-level regression test that verifies the home-page selector no longer exposes `tenant` or `endpoint`, while retaining `model` and `provider`.
- Run Prettier on every modified Vue and JavaScript file.
- Run the focused regression test and the frontend production build.
- Run `git diff --check` to detect whitespace errors.
