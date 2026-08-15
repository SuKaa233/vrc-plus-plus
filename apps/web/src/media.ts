export interface AvatarSource {
  userIcon?: string
  imageUrl?: string
  currentAvatarThumbnailImageUrl?: string
}

/**
 * VRChat userIcon is commonly returned as a versioned file URL. Requesting the
 * image rendition keeps the user's chosen icon while avoiding the original,
 * potentially multi-megabyte asset.
 */
export function optimizedVrcImageUrl(value?: string, size = 256) {
  if (!value) return ''
  try {
    const url = new URL(value)
    if (url.hostname !== 'api.vrchat.cloud') return value
    const match = url.pathname.match(/^\/api\/1\/file\/(file_[^/]+)\/(\d+)(?:\/file)?\/?$/)
    if (!match) return value
    url.pathname = `/api/1/image/${match[1]}/${match[2]}/${size}`
    url.search = ''
    url.hash = ''
    return url.toString()
  } catch {
    return value
  }
}

export function preferredFriendAvatar(friend?: AvatarSource | null) {
  return optimizedVrcImageUrl(friend?.userIcon)
    || friend?.currentAvatarThumbnailImageUrl
    || optimizedVrcImageUrl(friend?.imageUrl)
}
