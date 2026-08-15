const [webSocketUrl] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-features-1-4.mjs <ws-url>')

const socket = new WebSocket(webSocketUrl)
let id = 0
const pending = new Map()
const command = (method, params = {}) => new Promise((resolve, reject) => {
  const requestId = ++id
  const timer = setTimeout(() => {
    pending.delete(requestId)
    reject(new Error(`${method} timed out`))
  }, 10000)
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
await new Promise((resolve, reject) => {
  socket.onopen = resolve
  socket.onerror = reject
})
await command('Runtime.enable')
await command('Page.enable')
await command('Page.reload', { ignoreCache: true })
await new Promise((resolve) => setTimeout(resolve, 1800))

const evaluate = async (expression) => {
  const result = await command('Runtime.evaluate', {
    expression,
    awaitPromise: true,
    returnByValue: true,
  })
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text)
  return result.result.value
}
const wait = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds))
const clickNavigation = async (label) => {
  await evaluate(`(() => {
    const target = [...document.querySelectorAll('button, a')].find((button) => button.textContent?.includes(${JSON.stringify(label)}))
    target?.click()
    return Boolean(target)
  })()`)
  await wait(900)
}

await clickNavigation('关系网')
const networkInitial = await evaluate(`(() => ({
  communities: document.querySelectorAll('.community-item').length,
  nodes: document.querySelectorAll('.graph-node').length,
  edges: document.querySelectorAll('.network-edge').length,
  tags: Math.max(0, (document.querySelector('.network-toolbar select')?.options.length || 1) - 1),
  pathControls: document.querySelectorAll('.path-toolbar select').length,
}))()`)

const collapse = await evaluate(`(async () => {
  const item = [...document.querySelectorAll('.community-item')].find((entry) => {
    const match = entry.textContent?.match(/(\\d+)\\s*(?:位|人)/)
    return Number(match?.[1] || 0) >= 2
  })
  const button = item && [...item.querySelectorAll('button')].find((entry) => /折叠|展开/.test(entry.textContent || ''))
  if (!button) return { supported: false }
  const before = document.querySelectorAll('.graph-node').length
  button.click()
  await new Promise((resolve) => setTimeout(resolve, 700))
  const after = document.querySelectorAll('.graph-node').length
  const restore = [...item.querySelectorAll('button')].find((entry) => /折叠|展开/.test(entry.textContent || ''))
  restore?.click()
  await new Promise((resolve) => setTimeout(resolve, 700))
  return { supported: true, before, after, restored: document.querySelectorAll('.graph-node').length }
})()`)

const path = await evaluate(`(async () => {
  const edge = document.querySelector('.network-edge')
  const [start, end] = (edge?.getAttribute('data-edge-key') || '').split('|')
  const selects = [...document.querySelectorAll('.path-toolbar select')]
  if (!start || !end || selects.length !== 2) return { supported: false }
  for (const [select, value] of [[selects[0], start], [selects[1], end]]) {
    select.value = value
    select.dispatchEvent(new Event('change', { bubbles: true }))
  }
  await new Promise((resolve) => setTimeout(resolve, 100))
  const find = [...document.querySelectorAll('.path-toolbar button')].find((button) => button.textContent?.includes('查找'))
  find?.click()
  await new Promise((resolve) => setTimeout(resolve, 250))
  const result = {
    supported: true,
    highlightedEdges: document.querySelectorAll('.path-edge').length,
    highlightedNodes: document.querySelectorAll('.path-node').length,
    message: document.querySelector('.path-toolbar small')?.textContent?.trim() || '',
  }
  const clear = [...document.querySelectorAll('.path-toolbar button')].find((button) => button.textContent?.includes('清除'))
  clear?.click()
  return result
})()`)

const crossCommunity = await evaluate(`(async () => {
  const button = [...document.querySelectorAll('.network-toolbar button')].find((entry) => entry.textContent?.includes('跨圈'))
  if (!button) return { supported: false }
  const before = document.querySelectorAll('.network-edge').length
  button.click()
  await new Promise((resolve) => setTimeout(resolve, 700))
  const after = document.querySelectorAll('.network-edge').length
  button.click()
  await new Promise((resolve) => setTimeout(resolve, 700))
  return { supported: true, before, after, restored: document.querySelectorAll('.network-edge').length }
})()`)

const tagFilter = await evaluate(`(async () => {
  const select = [...document.querySelectorAll('.network-toolbar select')].find((entry) => [...entry.options].some((option) => option.textContent?.includes('标签')))
  const option = select && [...select.options].find((entry) => entry.value)
  if (!select || !option) return { supported: false }
  const before = document.querySelectorAll('.graph-node').length
  select.value = option.value
  select.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise((resolve) => setTimeout(resolve, 700))
  const after = document.querySelectorAll('.graph-node').length
  select.value = ''
  select.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise((resolve) => setTimeout(resolve, 700))
  return { supported: true, tag: option.textContent, before, after, restored: document.querySelectorAll('.graph-node').length }
})()`)

await clickNavigation('好友')
const friendList = await evaluate(`(async () => {
  const list = document.querySelector('.virtual-friend-list')
  const before = [...document.querySelectorAll('.virtual-friend')].map((item) => item.textContent?.trim()).slice(0, 2)
  const total = document.querySelector('.workbench-heading h2')?.textContent?.trim() || ''
  const renderedBefore = document.querySelectorAll('.virtual-friend').length
  if (list) {
    list.scrollTop = 1000
    list.dispatchEvent(new Event('scroll'))
    await new Promise((resolve) => setTimeout(resolve, 150))
  }
  const after = [...document.querySelectorAll('.virtual-friend')].map((item) => item.textContent?.trim()).slice(0, 2)
  const result = {
    total,
    rendered: renderedBefore,
    columns: getComputedStyle(document.querySelector('.virtual-friend') || document.body).width,
    filterSelects: document.querySelectorAll('.friend-filters select').length,
    queryInputs: document.querySelectorAll('.friend-filters input').length,
    scrollWindowChanged: JSON.stringify(before) !== JSON.stringify(after),
  }
  if (list) {
    list.scrollTop = 0
    list.dispatchEvent(new Event('scroll'))
  }
  return result
})()`)

socket.close()
process.stdout.write(JSON.stringify({ networkInitial, collapse, path, crossCommunity, tagFilter, friendList }, null, 2))
