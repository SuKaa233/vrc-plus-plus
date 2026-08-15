import fs from 'node:fs'

const [webSocketUrl, output, view = '历史', widthValue = '1600', heightValue = '1000', itemText = '', actionText = ''] = process.argv.slice(2)
if (!webSocketUrl || !output) throw new Error('usage: node capture-ui.mjs <ws-url> <output.png> [view] [width] [height]')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  pending.set(requestId, { resolve, reject, method })
  socket.send(JSON.stringify({ id: requestId, method, params }))
})
socket.onmessage = ({ data }) => {
  const message = JSON.parse(data)
  if (!message.id || !pending.has(message.id)) return
  const task = pending.get(message.id)
  pending.delete(message.id)
  if (message.error) task.reject(new Error(`${task.method}: ${message.error.message}`))
  else task.resolve(message.result)
}
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject })
await command('Page.enable')
await command('Runtime.enable')
try {
  await command('Emulation.setDeviceMetricsOverride', { width: Math.trunc(Number(widthValue) || 1600), height: Math.trunc(Number(heightValue) || 1000), deviceScaleFactor: 1, mobile: false })
} catch {
  // Older Edge builds can reject metrics overrides; capture the user's existing viewport instead.
}
await command('Page.reload', { ignoreCache: true })
await new Promise((resolve) => setTimeout(resolve, 2500))
await command('Runtime.evaluate', { expression: `(() => { const button = [...document.querySelectorAll('button')].find((item) => item.textContent?.trim().includes(${JSON.stringify(view)})); if (button) button.click(); return Boolean(button) })()` })
await new Promise((resolve) => setTimeout(resolve, 7500))
if (itemText && itemText !== '-') {
  await command('Runtime.evaluate', { expression: `(() => { const button = [...document.querySelectorAll('.world-card')].find((item) => item.textContent?.includes(${JSON.stringify(itemText)})); if (button) button.click(); return Boolean(button) })()` })
  await new Promise((resolve) => setTimeout(resolve, 4500))
}
if (actionText) {
  await command('Runtime.evaluate', { expression: `(() => { const button = [...document.querySelectorAll('button')].find((item) => item.textContent?.trim().includes(${JSON.stringify(actionText)})); if (button) button.click(); return Boolean(button) })()` })
  await new Promise((resolve) => setTimeout(resolve, 900))
}
const result = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
fs.writeFileSync(output, Buffer.from(result.data, 'base64'))
socket.close()
