const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-network-state.mjs <ws-url>')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => { pending.delete(requestId); reject(new Error(`${method} timed out`)) }, 8000)
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
const result = await command('Runtime.evaluate', {
  expression: `(() => {
    const stage = document.querySelector('.graph-stage')
    const svg = stage?.querySelector(':scope > svg')
    const viewport = svg?.querySelector('.graph-viewport')
    const stageRect = stage?.getBoundingClientRect()
    const points = stageRect ? [
      [stageRect.left + stageRect.width / 2, stageRect.top + 20],
      [stageRect.left + stageRect.width / 2, stageRect.top + stageRect.height / 2],
      [stageRect.right - 30, stageRect.top + 30],
    ] : []
    const describe = (element) => element ? {
      tag: element.tagName,
      className: typeof element.className === 'string' ? element.className : element.className?.baseVal,
      id: element.id,
      position: getComputedStyle(element).position,
      zIndex: getComputedStyle(element).zIndex,
      background: getComputedStyle(element).backgroundColor,
      opacity: getComputedStyle(element).opacity,
      pointerEvents: getComputedStyle(element).pointerEvents,
    } : null
    return {
      url: location.href,
      stage: describe(stage),
      stageRect: stageRect && { x: stageRect.x, y: stageRect.y, width: stageRect.width, height: stageRect.height },
      svg: describe(svg),
      svgRect: svg && { x: svg.getBoundingClientRect().x, y: svg.getBoundingClientRect().y, width: svg.getBoundingClientRect().width, height: svg.getBoundingClientRect().height },
      viewportTransform: viewport?.getAttribute('transform'),
      nodes: svg?.querySelectorAll('.graph-node').length || 0,
      images: svg?.querySelectorAll('image').length || 0,
      muted: svg?.querySelectorAll('.graph-node.muted').length || 0,
      hovered: svg?.querySelector('.graph-node.hovered title')?.textContent || '',
      communities: [...document.querySelectorAll('.community-item')].map((item) => item.textContent?.trim()),
      renderedEdges: svg?.querySelectorAll('.network-edge').length || 0,
      pathControls: document.querySelectorAll('.path-toolbar select').length,
      controls: describe(stage?.querySelector('.graph-controls')),
      controlsRect: (() => { const rect = stage?.querySelector('.graph-controls')?.getBoundingClientRect(); return rect && { x: rect.x, y: rect.y, width: rect.width, height: rect.height } })(),
      hitStack: points.map(([x, y]) => ({ x, y, elements: document.elementsFromPoint(x, y).slice(0, 8).map(describe) })),
      fixedLayers: [...document.querySelectorAll('body *')].filter((element) => getComputedStyle(element).position === 'fixed' && element.getBoundingClientRect().width > innerWidth * .5).map(describe),
    }
  })()`,
  returnByValue: true,
})
socket.close()
process.stdout.write(JSON.stringify(result.result.value, null, 2))
process.exit(0)
