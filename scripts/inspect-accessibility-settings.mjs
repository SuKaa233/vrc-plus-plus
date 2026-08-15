const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-accessibility-settings.mjs <ws-url>')
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
  message.error ? task.reject(new Error(message.error.message)) : task.resolve(message.result)
}
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject })
await command('Runtime.enable')
const evaluate = async (expression) => {
  const result = await command('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text)
  return result.result.value
}
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

const openedProfile = await evaluate(`(()=>{const button=document.querySelector('.sidebar-user-main');button?.click();return Boolean(button)})()`)
await wait(2500)
const profile = await evaluate(`(() => ({
  opened:Boolean(document.querySelector('.profile-dialog')),
  title:document.querySelector('.profile-dialog h2')?.textContent?.trim(),
  fields:[...document.querySelectorAll('.profile-dialog label>span:first-child,.profile-dialog legend')].map(x=>x.textContent?.trim()),
  saveLabel:document.querySelector('.profile-dialog footer button')?.textContent?.trim(),
  error:document.querySelector('.profile-error')?.textContent?.trim()||''
}))()`)
await evaluate(`document.querySelector('.profile-close')?.click()`)
const openedDisplay = await evaluate(`(()=>{const button=[...document.querySelectorAll('.sidebar-bottom>button')].find(x=>x.textContent?.includes('字体大小'));button?.click();return Boolean(button)})()`)
await wait(300)
const display = await evaluate(`(() => ({
  opened:Boolean(document.querySelector('.display-settings-modal')),
  scale:document.querySelector('.font-scale-control input')?.value,
  language:document.querySelector('.language-setting span')?.textContent?.trim(),
  bodyWidth:document.body.scrollWidth,
  viewport:innerWidth
}))()`)
const scaleRegression = await evaluate(`(async()=>{const input=document.querySelector('.font-scale-control input');if(!input)return null;const values=[];for(const value of ['90','115','140']){input.value=value;input.dispatchEvent(new Event('input',{bubbles:true}));await new Promise(r=>setTimeout(r,80));values.push({value,bodyWidth:document.body.scrollWidth,viewport:innerWidth,consoleRight:Math.round(document.querySelector('.console-main')?.getBoundingClientRect().right||0)})}input.value='115';input.dispatchEvent(new Event('input',{bubbles:true}));return values})()`)
await evaluate(`document.querySelector('.display-settings-modal .icon-button')?.click()`)
const language = await evaluate(`(async()=>{const button=[...document.querySelectorAll('.sidebar-bottom>button')].find(x=>x.textContent?.includes('English'));if(!button)return {found:false};button.click();await new Promise(r=>setTimeout(r,150));const englishTitle=document.querySelector('.console-header h1')?.textContent?.trim();const back=[...document.querySelectorAll('.sidebar-bottom>button')].find(x=>x.textContent?.includes('简体中文'));back?.click();return {found:true,englishTitle,restored:Boolean(back)}})()`)
socket.close()
console.log(JSON.stringify({ openedProfile, profile, openedDisplay, display, scaleRegression, language }, null, 2))
