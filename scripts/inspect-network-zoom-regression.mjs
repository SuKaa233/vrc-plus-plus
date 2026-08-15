import fs from 'node:fs'
import path from 'node:path'

const [webSocketUrl, outputDirectory] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-network-zoom-regression.mjs <ws-url> [output-dir]')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => { pending.delete(requestId); reject(new Error(`${method} timed out`)) }, 25000)
  pending.set(requestId, { resolve: (value) => { clearTimeout(timer); resolve(value) }, reject })
  socket.send(JSON.stringify({ id: requestId, method, params }))
})
socket.onmessage = ({ data }) => {
  const message = JSON.parse(data)
  const task = pending.get(message.id)
  if (!task) return
  pending.delete(message.id)
  message.error ? task.reject(new Error(message.error.message)) : task.resolve(message.result)
}
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject })
await command('Runtime.enable')
await command('Page.enable')
await command('Page.reload', { ignoreCache: true })
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
const evaluate = async (expression) => {
  const result = await command('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text)
  return result.result.value
}
await wait(3500)
await evaluate(`(()=>{[...document.querySelectorAll('.sidebar nav button')].find(x=>x.textContent?.trim().startsWith('关系网'))?.click()})()`)
await wait(5000)
if (outputDirectory) fs.mkdirSync(outputDirectory, { recursive: true })
const capture = async (name) => {
  if (!outputDirectory) return
  const result = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
  fs.writeFileSync(path.join(outputDirectory, `${name}.png`), Buffer.from(result.data, 'base64'))
}
const setZoom = async (direction, clicks) => {
  await evaluate(`(()=>{document.querySelectorAll('.graph-controls button')[2]?.click();const button=document.querySelectorAll('.graph-controls button')[${direction === 'out' ? 0 : 1}];for(let i=0;i<${clicks};i++)button?.click()})()`)
  await wait(450)
}
const focusCore = async () => evaluate(`(()=>{const degree=node=>Number(node.querySelector('title')?.textContent?.split('·')[1]?.trim().split(' ')[0]||0);const node=[...document.querySelectorAll('.graph-node')].sort((a,b)=>degree(b)-degree(a))[0];node?.dispatchEvent(new PointerEvent('pointerenter'));return node?.querySelector('title')?.textContent||''})()`)
const readState = async () => evaluate(`(()=>({zoom:document.querySelector('.graph-controls span')?.textContent?.trim(),nodes:document.querySelectorAll('.graph-node').length,images:document.querySelectorAll('.node-avatar').length,circularDataImages:[...document.querySelectorAll('.node-avatar')].filter(x=>x.getAttribute('href')?.startsWith('data:image/')).length,clipPaths:document.querySelectorAll('clipPath,mask').length,focused:document.querySelector('.graph-stage')?.classList.contains('focused')||false,highlighted:document.querySelectorAll('.network-edge.highlighted').length,errors:[...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),bodyWidth:document.body.scrollWidth,viewport:innerWidth}))()`)
const results = []
for (const item of [{ name: '080', direction: 'out', clicks: 1 }, { name: '100', direction: 'in', clicks: 0 }, { name: '125', direction: 'in', clicks: 1 }, { name: '195', direction: 'in', clicks: 3 }]) {
  await setZoom(item.direction, item.clicks)
  const target = await focusCore()
  await wait(180)
  results.push({ target, ...(await readState()) })
  await capture(`zoom-${item.name}`)
}
await evaluate(`(()=>{document.querySelectorAll('.graph-controls button')[2]?.click();document.querySelector('.graph-node.hovered')?.dispatchEvent(new PointerEvent('pointerleave'))})()`)
socket.close()
console.log(JSON.stringify(results, null, 2))
