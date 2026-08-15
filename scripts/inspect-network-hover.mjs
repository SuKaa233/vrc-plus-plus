import fs from 'node:fs'

const [webSocketUrl, output, zoomClicksValue = '0', degreeMode = 'min'] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-network-hover.mjs <ws-url> [output.png] [zoom-clicks] [min|max]')
const zoomClicks = Math.max(0, Math.min(5, Math.trunc(Number(zoomClicksValue) || 0)))
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => {
    pending.delete(requestId)
    reject(new Error(`${method} timed out`))
  }, 12000)
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
await command('Runtime.enable')
await command('Page.enable')
if (zoomClicks) {
  await command('Runtime.evaluate', {
    expression: `(() => { const plus = document.querySelectorAll('.graph-controls button')[1]; for (let index = 0; index < ${zoomClicks}; index += 1) plus?.click() })()`,
  })
  await new Promise((resolve) => setTimeout(resolve, 500))
}
const candidate = await command('Runtime.evaluate', {
  expression: `(() => {
    const candidates = []
    for (const [index, node] of [...document.querySelectorAll('.graph-node')].entries()) {
      const rect = node.getBoundingClientRect()
      if (rect.width < 8 || rect.height < 8) continue
      const x = rect.left + rect.width / 2
      const y = rect.top + rect.height / 2
      if (x < 0 || y < 0 || x > innerWidth || y > innerHeight) continue
      const title = node.querySelector('title')?.textContent || ''
      candidates.push({ index, x, y, title, degree: Number(title.split('·')[1]?.trim().split(' ')[0] || 999) })
    }
    return candidates.sort((left, right) => ${degreeMode === 'max' ? 'right.degree - left.degree' : 'left.degree - right.degree'})[0] || null
  })()`,
  returnByValue: true,
})
if (!candidate.result.value) throw new Error('no visible relationship node found')
const { index: nodeIndex } = candidate.result.value
const samples = []
for (let index = 0; index < 16; index += 1) {
  await command('Runtime.evaluate', {
    expression: `(() => {
      const node = document.querySelectorAll('.graph-node')[${nodeIndex}]
      if (!node) return false
      if (${index} > 0) node.dispatchEvent(new PointerEvent('pointerleave'))
      node.dispatchEvent(new PointerEvent('pointerenter'))
      return true
    })()`,
  })
  await new Promise((resolve) => setTimeout(resolve, 45))
  const state = await command('Runtime.evaluate', {
    expression: `({
      focused: document.querySelector('.graph-stage')?.classList.contains('focused') || false,
      hovered: document.querySelector('.graph-node.hovered title')?.textContent || '',
      highlightedLines: document.querySelectorAll('line.highlighted').length,
      relatedNodes: document.querySelectorAll('.graph-node.related').length,
      dimmedNodes: [...document.querySelectorAll('.graph-node')].filter((node) => Number(getComputedStyle(node).opacity) < 1).length,
      visibleLabels: [...document.querySelectorAll('.node-label')].filter((label) => getComputedStyle(label).display !== 'none').length,
      relatedLabels: document.querySelectorAll('.graph-node.related .node-label').length,
      currentHaloOpacity: getComputedStyle(document.querySelector('.graph-node.hovered .node-focus-halo') || document.documentElement).opacity,
      relatedHaloOpacity: getComputedStyle(document.querySelector('.graph-node.related:not(.hovered) .node-focus-halo') || document.documentElement).opacity,
      highlightedLineOpacity: getComputedStyle(document.querySelector('line.highlighted') || document.documentElement).opacity,
      backgroundLineOpacity: getComputedStyle(document.querySelector('.graph-viewport > line:not(.highlighted)') || document.documentElement).opacity,
    })`,
    returnByValue: true,
  })
  samples.push(state.result.value)
}
const boundarySamples = []
const secondNodeIndex = nodeIndex === 0 ? 1 : 0
for (let index = 0; index < 12; index += 1) {
  const fromIndex = index % 2 === 0 ? nodeIndex : secondNodeIndex
  const toIndex = index % 2 === 0 ? secondNodeIndex : nodeIndex
  await command('Runtime.evaluate', {
    expression: `(() => {
      const nodes = document.querySelectorAll('.graph-node')
      nodes[${fromIndex}]?.dispatchEvent(new PointerEvent('pointerleave'))
      nodes[${toIndex}]?.dispatchEvent(new PointerEvent('pointerenter'))
    })()`,
  })
  await new Promise((resolve) => setTimeout(resolve, 35))
  const state = await command('Runtime.evaluate', {
    expression: `document.querySelector('.graph-node.hovered title')?.textContent || ''`,
    returnByValue: true,
  })
  boundarySamples.push(state.result.value)
}
if (output) {
  const shot = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
  fs.writeFileSync(output, Buffer.from(shot.data, 'base64'))
}
if (zoomClicks) {
  await command('Runtime.evaluate', {
    expression: `(() => { const minus = document.querySelectorAll('.graph-controls button')[0]; for (let index = 0; index < ${zoomClicks}; index += 1) minus?.click() })()`,
  })
}
await command('Runtime.evaluate', {
  expression: `(() => { document.querySelector('.graph-node.hovered')?.dispatchEvent(new PointerEvent('pointerleave')) })()`,
})
await new Promise((resolve) => setTimeout(resolve, 140))
socket.close()
process.stdout.write(JSON.stringify({ candidate: candidate.result.value, samples, boundarySamples }, null, 2))
process.exit(0)
