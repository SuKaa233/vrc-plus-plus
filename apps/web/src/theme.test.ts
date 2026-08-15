import { describe, expect, it } from 'vitest'
import { resolveTheme } from './theme'

describe('resolveTheme', () => {
  it('prefers a saved theme over the system setting', () => {
    expect(resolveTheme('dark', true)).toBe('dark')
    expect(resolveTheme('light', false)).toBe('light')
  })

  it('uses the system preference when no valid choice was saved', () => {
    expect(resolveTheme(null, true)).toBe('light')
    expect(resolveTheme('unexpected', false)).toBe('dark')
  })
})
