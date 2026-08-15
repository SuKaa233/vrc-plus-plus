const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node measure-network-density.mjs <ws-url>')

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
  expression: `(() => {
    const nodes = [...document.querySelectorAll('.graph-node')].map((element) => {
      const match = element.getAttribute('transform')?.match(/translate\\(([-.\\d]+)[ ,]+([-.\\d]+)\\)/)
      return {
        x: Number(match?.[1]),
        y: Number(match?.[2]),
        radius: Number(element.querySelector('.node-body')?.getAttribute('r')),
      }
    }).filter((node) => Number.isFinite(node.x) && Number.isFinite(node.y) && Number.isFinite(node.radius))
    let avatarOverlaps = 0
    let ringOverlaps = 0
    let minimumAvatarGap = Infinity
    const clearances = []
    for (let left = 0; left < nodes.length; left += 1) {
      for (let right = left + 1; right < nodes.length; right += 1) {
        const a = nodes[left]
        const b = nodes[right]
        const distance = Math.hypot(a.x - b.x, a.y - b.y)
        const avatarGap = distance - a.radius - b.radius
        clearances.push(avatarGap)
        minimumAvatarGap = Math.min(minimumAvatarGap, avatarGap)
        if (avatarGap < 0) avatarOverlaps += 1
        if (distance < a.radius + b.radius + 10) ringOverlaps += 1
      }
    }
    clearances.sort((a, b) => a - b)
    const labels = [...document.querySelectorAll('.node-label')].map((label) => label.getBoundingClientRect()).filter((rect) => rect.width && rect.height)
    let labelOverlaps = 0
    for (let left = 0; left < labels.length; left += 1) {
      for (let right = left + 1; right < labels.length; right += 1) {
        const a = labels[left]
        const b = labels[right]
        if (a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top) labelOverlaps += 1
      }
    }
    return {
      nodes: nodes.length,
      labels: labels.length,
      avatarOverlaps,
      ringOverlaps,
      labelOverlaps,
      minimumAvatarGap: Number(minimumAvatarGap.toFixed(2)),
      p01AvatarGap: Number((clearances[Math.floor(clearances.length * .01)] ?? 0).toFixed(2)),
    }
  })()`,
  returnByValue: true,
})
socket.close()
process.stdout.write(JSON.stringify(result.result.value, null, 2))
