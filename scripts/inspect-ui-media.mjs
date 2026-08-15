const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-ui-media.mjs <ws-url>')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  pending.set(requestId, { resolve, reject })
  socket.send(JSON.stringify({ id: requestId, method, params }))
})
socket.onmessage = ({ data }) => {
  const message = JSON.parse(data)
  const task = pending.get(message.id)
  if (!task) return
  pending.delete(message.id)
  if (message.error) task.reject(new Error(message.error.message))
  else task.resolve(message.result)
}
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject })
await command('Runtime.enable')
const result = await command('Runtime.evaluate', {
  expression: `Array.from(document.images).map((image) => ({
    className: image.className,
    complete: image.complete,
    naturalWidth: image.naturalWidth,
    loading: image.loading,
    display: getComputedStyle(image).display,
    opacity: getComputedStyle(image).opacity,
    visibility: getComputedStyle(image).visibility,
    rect: { width: image.getBoundingClientRect().width, height: image.getBoundingClientRect().height },
    src: image.currentSrc || image.src,
  }))`,
  returnByValue: true,
})
socket.close()
process.stdout.write(JSON.stringify(result.result.value, null, 2))
