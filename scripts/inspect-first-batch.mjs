import fs from 'node:fs'
import path from 'node:path'

const [webSocketUrl, outputDirectory] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-first-batch.mjs <ws-url>')
const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => { pending.delete(requestId); reject(new Error(`${method} timed out`)) }, 20000)
  pending.set(requestId, { resolve: (value) => { clearTimeout(timer); resolve(value) }, reject })
  socket.send(JSON.stringify({ id: requestId, method, params }))
})
socket.onmessage = ({ data }) => {
  const message = JSON.parse(data)
  const task = pending.get(message.id)
  if (!task) return
  pending.delete(message.id)
  if (message.error) task.reject(new Error(message.error.message)); else task.resolve(message.result)
}
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject })
await command('Runtime.enable'); await command('Page.enable'); await command('Page.reload', { ignoreCache: true })
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
await wait(2400)
const evaluate = async (expression) => {
  const result = await command('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text)
  return result.result.value
}
const clickNav = async (label) => {
  await evaluate(`(() => { const item=[...document.querySelectorAll('.sidebar nav button')].find(x=>x.textContent?.includes(${JSON.stringify(label)})); item?.click(); return Boolean(item) })()`)
  await wait(1000)
}
await evaluate(`(async()=>{for(let i=0;i<35;i++){const item=[...document.querySelectorAll('.sidebar nav button')].find(x=>x.textContent?.includes('好友'));if(Number(item?.querySelector('span')?.textContent||0)>0)return true;await new Promise(r=>setTimeout(r,300))}return false})()`)
const capture = async (name) => {
  if (!outputDirectory) return
  fs.mkdirSync(outputDirectory, { recursive: true })
  const result = await command('Page.captureScreenshot', { format: 'png', fromSurface: true, captureBeyondViewport: false })
  fs.writeFileSync(path.join(outputDirectory, `${name}.png`), Buffer.from(result.data, 'base64'))
}

const overview = await evaluate(`(() => ({
  statusCards: document.querySelectorAll('.today-status>div').length,
  joinable: document.querySelectorAll('.joinable-list>button').length,
  panels: document.querySelectorAll('.today-panel').length,
  errors: [...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),
}))()`)
await capture('first-batch-overview')

await clickNav('发现')
const discovery = await evaluate(`(async()=>{
  const input=document.querySelector('.discovery-hero input'); if(!input) return {supported:false}
  input.value='usr_d23865d0-9f0d-4933-8066-ab809c06d071'; input.dispatchEvent(new Event('input',{bubbles:true}));
  document.querySelector('.discovery-hero form')?.dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}));
  await new Promise(r=>setTimeout(r,2600));
  const sections=[...document.querySelectorAll('.discovery-section')]; const resultSection=sections.find(x=>x.querySelector('header strong')?.textContent?.includes('搜索结果')); const recentSection=sections.find(x=>x.querySelector('header strong')?.textContent?.includes('最近遇到'));
  return {supported:true,results:resultSection?.querySelectorAll('.person-row').length||0,recent:recentSection?.querySelectorAll('.person-row').length||0,labels:[...(resultSection?.querySelectorAll('.request-button')||[])].map(x=>x.textContent?.trim()).slice(0,5)}
})()`)
await capture('first-batch-discovery')
await clickNav('好友')
const profile = await evaluate(`(async()=>{
  const rows=[...document.querySelectorAll('.virtual-friend')]; const row=rows.find(x=>x.textContent?.includes('AxelTheFoxo'))||rows[0]; row?.click(); await new Promise(r=>setTimeout(r,3200));
  return {friendRows:document.querySelectorAll('.virtual-friend').length,rowFound:Boolean(row),open:Boolean(document.querySelector('.friend-detail')),insightCards:document.querySelectorAll('.insight-grid>div').length,timeline:document.querySelectorAll('.friend-timeline>div').length,insightText:document.querySelector('.local-insights')?.textContent?.replace(/\s+/g,' ').trim()||'',relation:[...document.querySelectorAll('.detail-actions button')].map(x=>x.textContent?.trim())}
})()`)
await capture('first-batch-profile')
await evaluate(`document.querySelector('.detail-toolbar button')?.click()`); await wait(300)

await clickNav('关系网')
const evolution = await evaluate(`(async()=>{
  const button=[...document.querySelectorAll('.network-toolbar button')].find(x=>x.textContent?.includes('朋友圈演变')); button?.click(); await new Promise(r=>setTimeout(r,400));
  return {supported:Boolean(button),blocks:document.querySelectorAll('.evolution-block').length,snapshots:document.querySelectorAll('.evolution-block select:first-of-type option').length,bridges:document.querySelectorAll('.bridge-list button').length,note:document.querySelector('.evolution-note')?.textContent?.trim()||''}
})()`)
await capture('first-batch-network')
console.log(JSON.stringify({ overview, discovery, profile, evolution }, null, 2))
socket.close()
