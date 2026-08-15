import fs from 'node:fs'

const [webSocketUrl, output] = process.argv.slice(2)
if (!webSocketUrl || !output) throw new Error('usage: node capture-current-ui.mjs <ws-url> <output.png>')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => {
    pending.delete(requestId)
    reject(new Error(`${method} timed out`))
  }, 15000)
  pending.set(requestId, {
    resolve: (value) => { clearTimeout(timer); resolve(value) },
    reject: (error) => { clearTimeout(timer); reject(error) },
  })
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
await command('Page.enable')
const result = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
fs.writeFileSync(output, Buffer.from(result.data, 'base64'))
socket.close()
process.exit(0)
