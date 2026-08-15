import fs from 'node:fs'
import path from 'node:path'

const [webSocketUrl, outputDirectory] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-fourth-phase.mjs <ws-url> [output-dir]')
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
await wait(4000)
const evaluate = async (expression) => {
  const result = await command('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text)
  return result.result.value
}
const capture = async (name) => {
  fs.mkdirSync(outputDirectory, { recursive: true })
  const result = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
  fs.writeFileSync(path.join(outputDirectory, `${name}.png`), Buffer.from(result.data, 'base64'))
}
const clickText = async (selector, label) => {
  const found = await evaluate(`(()=>{const item=[...document.querySelectorAll(${JSON.stringify(selector)})].find(x=>x.textContent?.trim().startsWith(${JSON.stringify(label)}));item?.click();return Boolean(item)})()`)
  if (!found) throw new Error(`missing ${label}`)
  await wait(900)
}

await clickText('.sidebar nav button', '旅程')
await wait(1200)
const tides = await evaluate(`(()=>({tabs:[...document.querySelectorAll('.journey-bar nav button')].map(x=>x.textContent?.trim()),cards:document.querySelectorAll('.tide-card').length,summary:document.querySelector('.section-summary')?.textContent?.replace(/\s+/g,' ').trim(),errors:[...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),bodyWidth:document.body.scrollWidth,viewport:window.innerWidth}))()`)
await capture('fourth-tides')
await clickText('.journey-bar nav button', '世界护照')
const passport = await evaluate(`(()=>({worlds:document.querySelectorAll('.passport-list>button').length,title:document.querySelector('.passport-detail h2')?.textContent?.trim()||'',facts:[...document.querySelectorAll('.passport-facts strong')].map(x=>x.textContent?.trim()),images:[...document.querySelectorAll('.passport-list img')].filter(x=>x.complete&&x.naturalWidth>0).length}))()`)
await capture('fourth-passport')
await clickText('.journey-bar nav button', '活动编队')
const plans = await evaluate(`(()=>({createFields:document.querySelectorAll('.plan-create input,.plan-create select').length,safety:document.querySelector('.plan-create small')?.textContent?.trim()||'',existing:document.querySelectorAll('.plan-board option').length}))()`)
await capture('fourth-plans')
await clickText('.journey-bar nav button', '起航检查')
const readiness = await evaluate(`(()=>({score:document.querySelector('.readiness-hero h2')?.textContent?.replace(/\s+/g,' ').trim(),checks:[...document.querySelectorAll('.check-card')].map(x=>({label:x.querySelector('strong')?.textContent?.trim(),state:x.dataset.state,detail:x.querySelector('p')?.textContent?.trim()})),errors:[...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),bodyWidth:document.body.scrollWidth,viewport:window.innerWidth}))()`)
await capture('fourth-readiness')
console.log(JSON.stringify({ tides, passport, plans, readiness }, null, 2))
socket.close()
