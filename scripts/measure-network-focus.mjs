const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node measure-network-focus.mjs <ws-url>')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => { pending.delete(requestId); reject(new Error(`${method} timed out`)) }, 12000)
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
await command('Page.bringToFront')
await command('Runtime.enable')
const result = await command('Runtime.evaluate', {
  expression: `(async () => {
    const nodes = [...document.querySelectorAll('.graph-node')].slice(0, 14)
    if (nodes.length < 2) return { error: 'not enough visible nodes' }
    const title = (node) => node.querySelector('title')?.textContent || ''
    nodes[0].dispatchEvent(new PointerEvent('pointerenter'))
    await new Promise((resolve) => setTimeout(resolve, 90))
    const timings = []
    let current = nodes[0]
    for (const next of nodes.slice(1)) {
      const started = performance.now()
      current.dispatchEvent(new PointerEvent('pointerleave'))
      next.dispatchEvent(new PointerEvent('pointerenter'))
      await new Promise((resolve) => {
        const check = () => {
          if (document.querySelector('.graph-node.hovered') === next || performance.now() - started > 500) resolve()
          else setTimeout(check, 5)
        }
        setTimeout(check, 5)
      })
      timings.push({ user: title(next), milliseconds: performance.now() - started, highlightedLines: document.querySelectorAll('.network-edge.highlighted').length })
      current = next
    }
    return {
      edgeCount: document.querySelectorAll('.network-edge').length,
      nodeCount: document.querySelectorAll('.graph-node').length,
      averageMilliseconds: timings.reduce((sum, item) => sum + item.milliseconds, 0) / timings.length,
      maximumMilliseconds: Math.max(...timings.map((item) => item.milliseconds)),
      timings,
    }
  })()`,
  awaitPromise: true,
  returnByValue: true,
})
socket.close()
process.stdout.write(JSON.stringify(result.result.value, null, 2))
process.exit(0)
