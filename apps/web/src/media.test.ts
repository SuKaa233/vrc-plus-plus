import { describe, expect, it } from 'vitest'
import { optimizedVrcImageUrl, preferredFriendAvatar } from './media'

describe('avatar media selection', () => {
  it('requests a lightweight rendition of a VRChat user icon', () => {
    expect(optimizedVrcImageUrl('https://api.vrchat.cloud/api/1/file/file_test-id/3')).toBe(
      'https://api.vrchat.cloud/api/1/image/file_test-id/3/256',
    )
    expect(optimizedVrcImageUrl('https://api.vrchat.cloud/api/1/file/file_test-id/3/file', 128)).toBe(
      'https://api.vrchat.cloud/api/1/image/file_test-id/3/128',
    )
  })

  it('leaves existing renditions and non-VRChat media untouched', () => {
    const thumbnail = 'https://api.vrchat.cloud/api/1/image/file_test-id/3/256'
    expect(optimizedVrcImageUrl(thumbnail)).toBe(thumbnail)
    expect(optimizedVrcImageUrl('https://cdn.example.com/avatar.png')).toBe('https://cdn.example.com/avatar.png')
  })

  it('prefers the custom user icon over the current avatar thumbnail', () => {
    expect(preferredFriendAvatar({
      userIcon: 'https://api.vrchat.cloud/api/1/file/file_custom/1',
      currentAvatarThumbnailImageUrl: 'https://api.vrchat.cloud/api/1/image/file_default-avatar/1/256',
    })).toBe('https://api.vrchat.cloud/api/1/image/file_custom/1/256')
  })
})
