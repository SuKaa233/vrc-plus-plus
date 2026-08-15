export const UI_SCALE_KEY = 'vrc-harbor-ui-scale'
export const DEFAULT_UI_SCALE = 1.15

export function normalizeUIScale(value: number) {
  if (!Number.isFinite(value)) return DEFAULT_UI_SCALE
  return Math.round(Math.min(1.4, Math.max(.9, value)) * 20) / 20
}

export function readUIScale(storage: Pick<Storage, 'getItem'> = localStorage) {
  const saved = Number(storage.getItem(UI_SCALE_KEY))
  return saved ? normalizeUIScale(saved) : DEFAULT_UI_SCALE
}

export function applyUIScale(value: number, persist = true, storage: Pick<Storage, 'setItem'> = localStorage) {
  const normalized = normalizeUIScale(value)
  document.documentElement.style.setProperty('--ui-scale', String(normalized))
  document.documentElement.dataset.uiScale = String(Math.round(normalized * 100))
  if (persist) storage.setItem(UI_SCALE_KEY, String(normalized))
  return normalized
}
