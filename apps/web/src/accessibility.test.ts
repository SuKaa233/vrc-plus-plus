import { describe, expect, it } from 'vitest'
import { DEFAULT_UI_SCALE, normalizeUIScale, readUIScale } from './accessibility'

describe('interface scale', () => {
  it('uses a more readable default and clamps unsafe values', () => {
    expect(readUIScale({ getItem: () => null })).toBe(DEFAULT_UI_SCALE)
    expect(normalizeUIScale(.4)).toBe(.9)
    expect(normalizeUIScale(2)).toBe(1.4)
    expect(normalizeUIScale(1.13)).toBe(1.15)
  })
})
