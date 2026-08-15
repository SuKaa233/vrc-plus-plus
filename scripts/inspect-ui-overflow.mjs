const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-ui-overflow.mjs <ws-url>')
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
  expression: `(() => ({
    viewport: { width: innerWidth, documentWidth: document.documentElement.scrollWidth },
    offenders: Array.from(document.querySelectorAll('*')).map((element) => {
      const rect = element.getBoundingClientRect()
      return { tag: element.tagName, className: String(element.className || ''), left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width), scrollWidth: element.scrollWidth }
    }).filter((item) => item.right > innerWidth + 1 || item.left < -1 || item.scrollWidth > item.width + 2).sort((left, right) => right.right - left.right).slice(0, 30),
  }))()`,
  returnByValue: true,
})
socket.close()
process.stdout.write(JSON.stringify(result.result.value, null, 2))
