import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocalApi } from './api'

afterEach(() => vi.unstubAllGlobals())

describe('LocalApi', () => {
  it('keeps the bootstrap CSRF token for write requests', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        appName: 'Test', version: '1', csrfToken: 'csrf-test', security: {},
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ status: 'anonymous' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }))
    vi.stubGlobal('fetch', fetchMock)

    const api = new LocalApi()
    await api.bootstrap()
    await api.login('user', 'password')

    const [, init] = fetchMock.mock.calls[1]
    expect((init.headers as Headers).get('X-CSRF-Token')).toBe('csrf-test')
    expect((init.headers as Headers).get('Content-Type')).toBe('application/json')
  })

  it('surfaces the local API error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { message: '线路暂不可用' },
    }), { status: 403, headers: { 'Content-Type': 'application/json' } })))
    await expect(new LocalApi().session()).rejects.toThrow('线路暂不可用')
  })

  it('encodes friend detail identifiers in local API paths', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      items: [], source: 'live', fetchedAt: new Date().toISOString(), stale: false,
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    const api = new LocalApi()
    await api.user('usr_friend')
    await api.mutualFriends('usr_friend')

    expect(fetchMock.mock.calls[0][0]).toBe('/local/v1/users/usr_friend')
    expect(fetchMock.mock.calls[1][0]).toBe('/local/v1/users/usr_friend/mutual-friends')
  })

  it('uses explicit local-only routes for annotations and media cache clearing', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        appName: 'Test', version: '1', csrfToken: 'csrf-local', security: {},
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValue(new Response(JSON.stringify({}), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }))
    vi.stubGlobal('fetch', fetchMock)

    const api = new LocalApi()
    await api.bootstrap()
    await api.updateFriendAnnotation('usr_friend', { note: '备注', group: '常玩', color: '#5E7CE2', tags: ['中文'] })
    await api.clearMediaCache()

    expect(fetchMock.mock.calls[1][0]).toBe('/local/v1/friend-annotations/usr_friend')
    expect(fetchMock.mock.calls[1][1].method).toBe('PUT')
    expect((fetchMock.mock.calls[1][1].headers as Headers).get('X-CSRF-Token')).toBe('csrf-local')
    expect(fetchMock.mock.calls[2][0]).toBe('/local/v1/cache/media')
    expect(fetchMock.mock.calls[2][1].method).toBe('DELETE')
  })

  it('keeps world, history and notification mutations behind CSRF-protected local routes', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ appName: 'Test', version: '1', csrfToken: 'csrf-flow', security: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValue(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const api = new LocalApi()
    await api.bootstrap()
    await api.saveWorldFavorite({ id: 'wrld_test', name: '测试世界' }, '桌游')
    await api.notificationAction('not_test', 'accept')
    await api.clearActivity()

    expect(fetchMock.mock.calls.slice(1).map((call) => call[0])).toEqual([
      '/local/v1/world-favorites/wrld_test',
      '/local/v1/notifications/not_test/accept',
      '/local/v1/activity',
    ])
    for (const [, init] of fetchMock.mock.calls.slice(1)) {
      expect((init.headers as Headers).get('X-CSRF-Token')).toBe('csrf-flow')
    }
  })

  it('sends boops through the CSRF-protected local gateway', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ appName: 'Test', version: '1', csrfToken: 'csrf-boop', security: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const api = new LocalApi()
    await api.bootstrap()
    await api.sendBoop('usr_friend', 'default_hand_wave')

    expect(fetchMock.mock.calls[1][0]).toBe('/local/v1/users/usr_friend/boop')
    expect(fetchMock.mock.calls[1][1].method).toBe('POST')
    expect(fetchMock.mock.calls[1][1].body).toBe(JSON.stringify({ emojiId: 'default_hand_wave' }))
    expect((fetchMock.mock.calls[1][1].headers as Headers).get('X-CSRF-Token')).toBe('csrf-boop')
  })
})
