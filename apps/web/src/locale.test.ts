import { describe, expect, it } from 'vitest'
import { interfaceText, readInterfaceLocale } from './locale'

describe('interface locale', () => {
  it('defaults to Chinese and can select English', () => {
    expect(readInterfaceLocale({ getItem: () => null })).toBe('zh-CN')
    expect(readInterfaceLocale({ getItem: () => 'en' })).toBe('en')
    expect(interfaceText('en', '好友', 'Friends')).toBe('Friends')
  })
})
