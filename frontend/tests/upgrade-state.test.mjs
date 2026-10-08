import test from 'node:test'
import assert from 'node:assert/strict'
import { createUpgradeController, isTerminalTask } from '../src/utils/upgrade-state.js'

const ok = (data) => Promise.resolve({ success: true, data })
const fail = (status, id) => Promise.resolve({ success: false, error: { code: status, id, detail: id || 'failed' } })

const deferred = () => {
    let resolve
    const promise = new Promise((done) => {
        resolve = done
    })
    return { promise, resolve }
}

const capabilityAllowed = { supported: true, allowed: true, reasons: [] }
const capabilityBlocked = { supported: true, allowed: false, reasons: ['upgrade_task_active'] }
const capabilityUnsupported = { supported: false, allowed: false, reasons: ['host_unsupported'] }
const preparation = {
    task_id: 'task-1',
    current_version: '1.0.0',
    target_version: 'v1.1.0',
    release_url: 'https://github.com/tokenlive/tokenlive-standalone/releases/tag/v1.1.0',
    confirm_expires_in: 300,
    credential: 'secret',
    restart_warning: true,
}
const taskQueued = { task_id: 'task-1', state: 'queued' }
const taskInstalling = { task_id: 'task-1', state: 'installing', phase: 'brew_upgrade' }
const taskSucceeded = { task_id: 'task-1', state: 'succeeded', target_version: 'v1.1.0' }
const taskFailed = {
    task_id: 'task-1',
    state: 'failed',
    failure_stage: 'brew_upgrade',
    error_kind: 'brew_upgrade_failed',
}

function setup(api, { timers, pollInterval } = {}) {
    const states = []
    const timeouts = []
    const controller = createUpgradeController({
        api,
        onChange: (state) => states.push(state),
        onSucceeded: () => states.push({ succeeded: true }),
        setTimeoutFn: timers ? (fn, ms) => timeouts.push(ms) : setTimeout,
        clearTimeoutFn: timers ? () => {} : clearTimeout,
        pollInterval: pollInterval ?? 2000,
    })
    return { controller, states, timeouts }
}

test('isTerminalTask classification', () => {
    assert.equal(isTerminalTask(taskQueued), false)
    assert.equal(isTerminalTask(taskSucceeded), true)
    assert.equal(isTerminalTask(taskFailed), true)
    assert.equal(isTerminalTask(null), false)
})

test('unauthorized controller never talks to the API', async () => {
    let capabilityCalls = 0
    const api = { capability: () => (capabilityCalls++, ok(capabilityAllowed)) }
    const { controller } = setup(api)
    controller.setAuthorized(false)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))
    assert.equal(capabilityCalls, 0)
    assert.equal(controller.getState().capability, null)
    controller.dispose()
})

test('activation loads capability and adopts an active task', async () => {
    const api = {
        capability: () => ok({ ...capabilityBlocked, active_task: taskInstalling }),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => ok(taskInstalling),
    }
    const { controller, states } = setup(api, { timers: true })
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))
    const last = states.at(-1)
    assert.equal(last.capability.supported, true)
    assert.equal(last.task.state, 'installing')
    controller.dispose()
})

test('prepare then confirm runs the double-confirmed flow', async () => {
    const seen = { prepare: null, submit: null }
    const api = {
        capability: () => ok(capabilityAllowed),
        prepare: (body) => {
            seen.prepare = body
            return ok(preparation)
        },
        submit: (body) => {
            seen.submit = body
            return ok(taskQueued)
        },
        task: () => ok(taskSucceeded),
    }
    const { controller, states } = setup(api, { timers: true })
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))

    await controller.prepare('v1.1.0')
    assert.deepEqual(seen.prepare, { target_version: 'v1.1.0' })
    assert.equal(controller.getState().preparation.task_id, 'task-1')

    await controller.confirm()
    assert.deepEqual(seen.submit, { task_id: 'task-1', credential: 'secret', confirm: true })
    assert.equal(controller.getState().preparation, null)
    assert.equal(controller.getState().task.state, 'queued')
    assert.ok(states.every((s) => !s.succeeded))
    controller.dispose()
})

test('unsupported capability never offers preparation', async () => {
    const api = {
        capability: () => ok(capabilityUnsupported),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => ok(taskSucceeded),
    }
    const { controller } = setup(api)
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))
    await controller.prepare('v1.1.0')
    assert.equal(controller.getState().preparation, null)
    controller.dispose()
})

test('terminal failure stops polling and keeps diagnostics', async () => {
    let taskCalls = 0
    const api = {
        capability: () => ok({ ...capabilityAllowed, active_task: taskQueued }),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => {
            taskCalls++
            return ok(taskFailed)
        },
    }
    const { controller } = setup(api, { pollInterval: 10 })
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 10))
    await new Promise((done) => setTimeout(done, 60))
    const pollsAfterFirst = taskCalls
    await new Promise((done) => setTimeout(done, 60))
    assert.equal(taskCalls, pollsAfterFirst, 'terminal task must stop polling')
    assert.equal(controller.getState().task.failure_stage, 'brew_upgrade')
    controller.dispose()
})

test('success refreshes capability and notifies the layout', async () => {
    const api = {
        capability: () => ok(capabilityAllowed),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => ok(taskSucceeded),
    }
    const { controller, states } = setup(api, { pollInterval: 10 })
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))
    await controller.prepare('v1.1.0')
    await controller.confirm()
    await new Promise((done) => setTimeout(done, 50))
    assert.ok(states.some((s) => s.succeeded === true))
    controller.dispose()
})

test('authorization loss clears state', async () => {
    const api = {
        capability: () => ok({ ...capabilityAllowed, active_task: taskQueued }),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => ok(taskQueued),
    }
    const { controller } = setup(api)
    controller.setAuthorized(true)
    controller.setActive(true)
    await new Promise((done) => setTimeout(done, 5))
    controller.setAuthorized(false)
    const state = controller.getState()
    assert.equal(state.capability, null)
    assert.equal(state.task, null)
    controller.dispose()
})

test('cancelPreparation drops the pending confirmation', async () => {
    const api = {
        capability: () => ok(capabilityAllowed),
        prepare: () => ok(preparation),
        submit: () => ok(taskQueued),
        task: () => ok(taskQueued),
    }
    const { controller } = setup(api)
    controller.setAuthorized(true)
    await controller.prepare('v1.1.0')
    assert.notEqual(controller.getState().preparation, null)
    controller.cancelPreparation()
    assert.equal(controller.getState().preparation, null)
    controller.dispose()
})
