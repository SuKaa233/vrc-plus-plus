import type { ActivityEvent, Friend, FriendNetwork, World } from './api'
import { detectNetworkCommunities, rankBridgeNodes } from './friend-network'

export interface InsightCard {
  id: string
  kind: 'route' | 'reunion' | 'pattern' | 'bridge' | 'coverage'
  title: string
  summary: string
  score: number
  reasons: string[]
  coverageDays: number
  generatedAt: string
  expiresAt: string
  targetUserId?: string
  targetWorldId?: string
  location?: string
}

export interface LocalQueryResult { intent: string; label: string; cards: InsightCard[]; hint?: string }
export type CompassMode = 'friends' | 'reunion' | 'explore'

function coverageDays(events: ActivityEvent[]) {
  return new Set(events.map((event) => event.observedAt.slice(0, 10)).filter(Boolean)).size
}

function timeBounds(hours = 1) {
  const generatedAt = new Date()
  return { generatedAt: generatedAt.toISOString(), expiresAt: new Date(generatedAt.getTime() + hours * 3600_000).toISOString() }
}

export function buildTonightCards(friends: Friend[], worlds: World[], favoriteIds: string[], events: ActivityEvent[], mode:CompassMode='friends'): InsightCard[] {
  const groups = new Map<string, Friend[]>()
  for (const friend of friends) {
    const location = friend.location ?? ''
    if (!friend.online || !location.startsWith('wrld_') || location.includes('~private(')) continue
    const items = groups.get(location) ?? []; items.push(friend); groups.set(location, items)
  }
  const worldByID = new Map(worlds.map((item) => [item.id, item]))
  const favorites = new Set(favoriteIds)
  const coverage = coverageDays(events)
  const lastSeenByUser = new Map<string,number>()
  for (const event of events) if (event.userId) lastSeenByUser.set(event.userId,Math.max(lastSeenByUser.get(event.userId)||0,new Date(event.observedAt).getTime()||0))
  const liveBounds = timeBounds(0.5)
  const live = [...groups.entries()].map(([location, people]) => {
    const worldID = location.split(':')[0]
    const world = worldByID.get(worldID)
    const favorite = favorites.has(worldID)
    const reunionBonus = mode==='reunion' ? Math.min(35,Math.max(...people.map(person=>Math.floor((Date.now()-(lastSeenByUser.get(person.id)||Date.now()))/86_400_000)),0)) : 0
    const exploreBonus = mode==='explore' ? Math.min(24,Math.round(Math.log10(Math.max(1,world?.visits||world?.favorites||1))*5)) : 0
    const livePriority = mode==='friends'?50:mode==='reunion'?35:0
    const score = livePriority + people.length * (mode==='friends'?30:20) + (favorite ? 14 : 0) + (world ? 6 : 0) + reunionBonus + exploreBonus
    const reasons = [`${people.length} 位好友位于同一可加入实例`, `包括 ${people.slice(0, 3).map((item) => item.displayName).join('、')}`]
    if (favorite) reasons.push('这个世界在你的 VRChat 收藏中')
    if (reunionBonus) reasons.push(`包含较久未见的在线好友，重逢优先级 +${reunionBonus}`)
    if (!world) reasons.push('世界资料尚未缓存，进入详情后可补充')
    return { id: `route:${location}`, kind: 'route' as const, title: world?.name || `${people.length} 位好友所在世界`, summary: people.length > 1 ? '实时可加入：熟人正在同一实例' : `实时可加入：可以去找 ${people[0].displayName}`, score, reasons, coverageDays: coverage, ...liveBounds, targetWorldId: worldID, location }
  }).sort((left, right) => right.score - left.score || left.title.localeCompare(right.title))

  const cards:InsightCard[] = live.slice(0,3)
  const usedWorlds = new Set(live.map(item=>item.targetWorldId))
  const history = new Map<string,{count:number;last:string}>()
  for (const event of events) {
    const worldID = event.worldId || (event.location?.startsWith('wrld_') ? event.location.split(':')[0] : '')
    if (!worldID) continue
    const current = history.get(worldID) || { count:0, last:event.observedAt }
    current.count += 1
    if (event.observedAt > current.last) current.last = event.observedAt
    history.set(worldID,current)
  }
  const fallbackBounds = timeBounds(12)
  const fallback = worlds.filter(world=>!usedWorlds.has(world.id)).map(world=>{
    const seen = history.get(world.id)
    const favorite = favorites.has(world.id)
    const popularity = Math.min(25,Math.round(Math.log10(Math.max(1,world.visits||world.favorites||1))*5))
    const score = (favorite?(mode==='explore'?48:38):0) + Math.min(36,(seen?.count||0)*(mode==='explore'?5:4)) + popularity + Math.min(15,world.occupants||0)
    const source = favorite&&seen?'收藏 + 本机历史':favorite?'收藏世界':seen?'本机历史':'热门公开世界'
    const reasons:string[] = []
    if (favorite) reasons.push('已在你的 VRChat 收藏中')
    if (seen) reasons.push(`本机记录出现 ${seen.count} 次，最近 ${new Date(seen.last).toLocaleDateString('zh-CN')}`)
    if (world.occupants) reasons.push(`当前公开占用人数约 ${world.occupants}`)
    if (!reasons.length) reasons.push(`热门度参考：${world.visits||world.favorites||0}`)
    reasons.push('这是世界建议，不代表有好友正在其中')
    return { id:`recommend:${world.id}`,kind:'route' as const,title:world.name,summary:`${source}推荐，可先查看世界与公开实例`,score,reasons,coverageDays:coverage,...fallbackBounds,targetWorldId:world.id }
  }).sort((a,b)=>b.score-a.score||a.title.localeCompare(b.title))
  if(mode==='explore') return [...live,...fallback].sort((a,b)=>b.score-a.score||a.title.localeCompare(b.title)).slice(0,6)
  for (const card of fallback) { if (cards.length>=6) break; cards.push(card) }
  return cards
}

export function buildReunionCards(friends: Friend[], events: ActivityEvent[], now = new Date(), minDays = 7): InsightCard[] {
  const stats = new Map<string, { count: number; last: Date }>()
  for (const event of events) {
    if (!event.userId || (!event.worldId && event.type !== 'game.player-joined' && event.type !== 'game.player-left')) continue
    const date = new Date(event.observedAt); if (Number.isNaN(date.getTime())) continue
    const item = stats.get(event.userId) ?? { count: 0, last: date }
    item.count += 1; if (date > item.last) item.last = date; stats.set(event.userId, item)
  }
  const coverage = coverageDays(events); const bounds = timeBounds(1)
  return friends.filter((friend) => friend.online && stats.has(friend.id)).map((friend) => {
    const stat = stats.get(friend.id)!
    const days = Math.max(0, Math.floor((now.getTime() - stat.last.getTime()) / 86_400_000))
    return { id: `reunion:${friend.id}`, kind: 'reunion' as const, title: friend.displayName, summary: `${days} 天未在本机记录中遇见，现在已上线`, score: Math.min(100, days * 2 + Math.min(stat.count, 20)), reasons: [`最近一次本机记录：${stat.last.toLocaleDateString('zh-CN')}`, `过去记录到 ${stat.count} 条相关事件`, friend.location?.startsWith('wrld_') ? '当前公开了可见世界' : '当前在线，位置未公开'], coverageDays: coverage, ...bounds, targetUserId: friend.id }
  }).filter((item) => Number(item.summary.split(' ')[0]) >= minDays).sort((left, right) => right.score - left.score).slice(0, 8)
}

function patternCards(events: ActivityEvent[], friends: Friend[], predicate: (date: Date) => boolean, label: string): InsightCard[] {
  const counts = new Map<string, number>()
  for (const event of events) { const date = new Date(event.observedAt); if (event.userId && (event.worldId || event.type === 'game.player-joined' || event.type === 'game.player-left') && predicate(date)) counts.set(event.userId, (counts.get(event.userId) ?? 0) + 1) }
  const friendByID = new Map(friends.map((item) => [item.id, item])); const coverage = coverageDays(events); const bounds = timeBounds(24)
  return [...counts.entries()].sort((a,b)=>b[1]-a[1]).slice(0,8).map(([id,count])=>({id:`pattern:${label}:${id}`,kind:'pattern',title:friendByID.get(id)?.displayName||id,summary:`${label}记录到 ${count} 条相关动态`,score:count,reasons:['来自实时好友动态与游戏记录','动态数量不等于共同会话次数'],coverageDays:coverage,...bounds,targetUserId:id}))
}

export function queryLocalSocial(query: string, friends: Friend[], worlds: World[], events: ActivityEvent[], network: FriendNetwork | null): LocalQueryResult {
  const text = query.trim().toLocaleLowerCase()
  if (/今晚|去哪|推荐|路线/.test(text)) return { intent:'tonight', label:'今晚可执行建议', cards:buildTonightCards(friends,worlds,[],events), hint:'优先实时可加入好友实例；不足三条时用本机历史和热门世界补足，并明确标注证据性质。' }
  if (/没见|重逢|很久|30\s*天/.test(text)) return { intent:'reunion', label:'很久没见但当前在线', cards:buildReunionCards(friends,events,new Date(),/30\s*天/.test(text)?30:7) }
  if (/周五|星期五|礼拜五/.test(text)) return { intent:'friday-night', label:'周五晚上的常遇好友', cards:patternCards(events,friends,(date)=>date.getDay()===5&&date.getHours()>=18,'周五晚上') }
  if (/跨圈|朋友圈|连接.*圈|桥梁/.test(text) && network) {
    const communities=detectNetworkCommunities(network.nodes,network.edges); const ranked=rankBridgeNodes(network.nodes,network.edges,communities); const coverage=coverageDays(events); const bounds=timeBounds(24)
    return {intent:'bridge',label:'连接多个朋友圈的好友',cards:ranked.slice(0,8).map((item)=>({id:`bridge:${item.id}`,kind:'bridge',title:network.nodes.find((node)=>node.id===item.id)?.displayName||item.id,summary:`连接 ${item.communityCount} 个朋友圈`,score:item.communityCount*20+item.crossEdges,reasons:[`${item.crossEdges} 条跨圈已观察连线`,'只基于当前本机关系网快照'],coverageDays:coverage,...bounds,targetUserId:item.id}))}
  }
  const namedFriend=friends.find((friend)=>text.includes(friend.displayName.toLocaleLowerCase())||text.includes(friend.id.toLocaleLowerCase()))
  if ((/世界|同场|常去|常见/.test(text)) && namedFriend) {
    const counts=new Map<string,number>(); for(const event of events)if(event.userId===namedFriend.id&&event.worldId)counts.set(event.worldId,(counts.get(event.worldId)??0)+1)
    const worldByID=new Map(worlds.map((world)=>[world.id,world]));const coverage=coverageDays(events);const bounds=timeBounds(24)
    return {intent:'friend-worlds',label:`与 ${namedFriend.displayName} 的常见世界`,cards:[...counts.entries()].sort((a,b)=>b[1]-a[1]).slice(0,8).map(([worldID,count])=>({id:`friend-world:${namedFriend.id}:${worldID}`,kind:'pattern',title:worldByID.get(worldID)?.name||worldID,summary:`本机记录到 ${count} 条共同相关事件`,score:count,reasons:[`好友：${namedFriend.displayName}`,'事件数量不等于完整到访次数'],coverageDays:coverage,...bounds,targetWorldId:worldID}))}
  }
  if (/覆盖|可信|数据|证据/.test(text)) {
    const days=coverageDays(events);const bounds=timeBounds(24);return {intent:'coverage',label:'本机数据覆盖',cards:[{id:'coverage',kind:'coverage',title:`覆盖 ${days} 个自然日`,summary:`当前查询使用 ${events.length} 条本机事件和 ${network?.edges.length??0} 条关系连线`,score:days,reasons:['断网与未运行时段没有观测','缺少记录不代表现实中没有活动'],coverageDays:days,...bounds}]}
  }
  return {intent:'help',label:'可以这样问',cards:[],hint:'试试“30 天没见但现在在线的好友”“周五晚上经常遇见谁”“连接多个朋友圈的人”“我和好友昵称最常在哪些世界同场”“数据覆盖怎么样”。'}
}
