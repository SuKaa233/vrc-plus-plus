import fs from 'node:fs'
import path from 'node:path'

const [webSocketUrl, outputDirectory] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-friend-motion.mjs <ws-url> [output-dir]')
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
const opened = await evaluate(`(()=>{const button=[...document.querySelectorAll('.sidebar nav button')].find(x=>x.textContent?.includes('旅程'));button?.click();return Boolean(button)})()`)
if (!opened) throw new Error('Journey navigation was not found')
await wait(15000)
const result = await evaluate(`(()=>({
  title:document.querySelector('.journey-bar h2')?.textContent?.trim(),
  tabs:[...document.querySelectorAll('.journey-bar nav button')].map(x=>x.textContent?.trim()),
  metrics:[...document.querySelectorAll('.motion-metrics>div')].map(x=>x.textContent?.trim()),
  liveScenes:document.querySelectorAll('.scene-card').length,
  intersections:document.querySelectorAll('.intersection-item').length,
  recentScenes:document.querySelectorAll('.recent-scene').length,
  bridgeButtons:[...document.querySelectorAll('.bridge-row button,.recent-scene footer button')].slice(0,12).map(x=>x.textContent?.trim()),
  loadedImages:[...document.querySelectorAll('.motion-view img')].filter(x=>x.complete&&x.naturalWidth>0).length,
  brokenImages:[...document.querySelectorAll('.motion-view img')].filter(x=>x.complete&&x.naturalWidth===0).length,
  placeholderScenes:[...document.querySelectorAll('.scene-title strong')].filter(x=>x.textContent?.trim()==='可见世界').length,
  intersectionNameFont:[...document.querySelectorAll('.intersection-names strong')].slice(0,2).map(x=>getComputedStyle(x).fontSize),
  errors:[...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),
  bodyWidth:document.body.scrollWidth,
  viewport:innerWidth,
  visibleText:document.querySelector('.motion-view')?.innerText?.slice(0,1800)||''
}))()`)
if (outputDirectory) {
  fs.mkdirSync(outputDirectory, { recursive: true })
  const screenshot = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
  fs.writeFileSync(path.join(outputDirectory, 'friend-motion.png'), Buffer.from(screenshot.data, 'base64'))
}
socket.close()
console.log(JSON.stringify(result, null, 2))
