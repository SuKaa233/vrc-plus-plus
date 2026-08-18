import type { ActivityEvent, Group, GroupMember, MutualFriend, UserProfile } from './api'

export type StrangerNodeKind = 'self' | 'target' | 'mutual' | 'group' | 'candidate'
export type StrangerEdgeKind = 'friend' | 'mutual' | 'membership' | 'candidate' | 'co_presence'
export interface StrangerGraphNode { id:string; label:string; kind:StrangerNodeKind; x:number; y:number; imageUrl?:string; detail:string }
export interface StrangerGraphEdge { source:string; target:string; kind:StrangerEdgeKind; label:string; evidence:'confirmed'|'observed' }
export interface StrangerGraph { nodes:StrangerGraphNode[]; edges:StrangerGraphEdge[]; hiddenNodes:number }
export interface RelationshipInsight { id:string; title:string; summary:string; confidence:'高'|'中'|'低'; evidence:string[]; tone:'strong'|'balanced'|'caution' }
export interface StrangerGraphInput {
  self:{ id:string; displayName:string; imageUrl?:string }
  target:UserProfile
  mutuals:MutualFriend[]
  groups:Group[]
  members:GroupMember[]
  events:ActivityEvent[]
}

function ring(index:number, total:number, cx:number, cy:number, rx:number, ry:number, start=-Math.PI/2) {
  const angle = start + index / Math.max(total, 1) * Math.PI * 2
  return { x:cx + Math.cos(angle)*rx, y:cy + Math.sin(angle)*ry }
}

function avatar(value:{ imageUrl?:string; profilePicOverride?:string; profilePicOverrideThumbnail?:string; currentAvatarThumbnailImageUrl?:string }) {
  return value.profilePicOverrideThumbnail || value.profilePicOverride || value.currentAvatarThumbnailImageUrl || value.imageUrl
}

function overlapsInInstance(left:ActivityEvent[], right:ActivityEvent[]) {
  return left.some(event => right.some(target => {
    const samePlace = Boolean(event.location && target.location && event.location === target.location)
    const close = Math.abs(new Date(event.observedAt).getTime() - new Date(target.observedAt).getTime()) <= 30*60*1000
    return samePlace && close
  }))
}

export function buildStrangerGraph(input:StrangerGraphInput):StrangerGraph {
  const nodes:StrangerGraphNode[] = [
    { id:input.self.id, label:input.self.displayName, kind:'self', x:115, y:290, imageUrl:input.self.imageUrl, detail:'你自己' },
    { id:input.target.id, label:input.target.displayName, kind:'target', x:470, y:290, imageUrl:avatar(input.target), detail:input.target.statusDescription || '目标陌生人' },
  ]
  const edges:StrangerGraphEdge[] = []
  if (input.target.isFriend) edges.push({ source:input.self.id, target:input.target.id, kind:'friend', label:'已是好友', evidence:'confirmed' })

  const mutuals = input.mutuals.slice(0, 18)
  mutuals.forEach((item,index) => {
    const point = ring(index, mutuals.length, 292, 290, 118, 225)
    nodes.push({ id:item.id, label:item.displayName, kind:'mutual', ...point, imageUrl:avatar(item), detail:'共同好友' })
    edges.push({ source:input.self.id, target:item.id, kind:'friend', label:'你的好友', evidence:'confirmed' })
    edges.push({ source:item.id, target:input.target.id, kind:'mutual', label:'共同好友', evidence:'confirmed' })
  })

  const groups = input.groups.slice(0, 6)
  groups.forEach((group,index) => {
    const point = ring(index, groups.length, 575, 290, 130, 230, -Math.PI/3)
    nodes.push({ id:group.id, label:group.name, kind:'group', ...point, imageUrl:group.iconUrl, detail:`公开群组${group.memberCount ? ` · ${group.memberCount} 人` : ''}` })
    edges.push({ source:input.target.id, target:group.id, kind:'membership', label:'公开群组', evidence:'confirmed' })
  })

  const groupIDs = new Set(groups.map(item => item.id))
  const memberGroups = new Map<string,Set<string>>()
  const memberByID = new Map<string,GroupMember>()
  for (const member of input.members) {
    if (!groupIDs.has(member.groupId || '')) continue
    memberByID.set(member.userId, member)
    const ids = memberGroups.get(member.userId) || new Set<string>()
    if (member.groupId) ids.add(member.groupId)
    memberGroups.set(member.userId, ids)
  }
  const candidates = [...memberByID.values()].filter(item => item.userId !== input.self.id && item.userId !== input.target.id && !input.mutuals.some(mutual => mutual.id === item.userId))
    .sort((a,b) => (memberGroups.get(b.userId)?.size || 0) - (memberGroups.get(a.userId)?.size || 0) || a.displayName.localeCompare(b.displayName)).slice(0,30)
  candidates.forEach((item,index) => {
    const point = ring(index, candidates.length, 800, 290, 175, 245)
    nodes.push({ id:item.userId, label:item.displayName, kind:'candidate', ...point, imageUrl:avatar(item), detail:`${memberGroups.get(item.userId)?.size || 1} 个共同公开群组` })
    for (const groupID of memberGroups.get(item.userId) || []) edges.push({ source:groupID, target:item.userId, kind:'candidate', label:'可见成员', evidence:'confirmed' })
  })

  const eventByUser = new Map<string,ActivityEvent[]>()
  for (const event of input.events) if (event.userId) eventByUser.set(event.userId, [...(eventByUser.get(event.userId) || []), event])
  const targetEvents = eventByUser.get(input.target.id) || []
  for (const candidate of candidates) {
    const matches = overlapsInInstance(eventByUser.get(candidate.userId) || [], targetEvents)
    if (matches) edges.push({ source:input.target.id, target:candidate.userId, kind:'co_presence', label:'日志同实例', evidence:'observed' })
  }
  for (const mutual of mutuals) {
    if (overlapsInInstance(eventByUser.get(mutual.id) || [], targetEvents)) edges.push({ source:input.target.id, target:mutual.id, kind:'co_presence', label:'共同好友同实例', evidence:'observed' })
  }
  const visible = new Set(nodes.map(item => item.id))
  return { nodes, edges:edges.filter(edge => visible.has(edge.source) && visible.has(edge.target)), hiddenNodes:Math.max(0,input.mutuals.length-mutuals.length)+Math.max(0,memberByID.size-candidates.length) }
}

export function relationshipInsights(input:StrangerGraphInput):RelationshipInsight[] {
  const mutualCount = Math.max(input.target.mutualFriendCount || 0, input.mutuals.length)
  const groupCount = Math.max(input.target.mutualGroupCount || 0, input.groups.length)
  const targetEvents = input.events.filter(item => item.userId === input.target.id)
  const groupFrequency = new Map<string,number>()
  for (const member of input.members) groupFrequency.set(member.userId, (groupFrequency.get(member.userId) || 0) + 1)
  const bridge = [...groupFrequency.entries()].sort((a,b)=>b[1]-a[1])[0]
  const profileFields = [input.target.bio,input.target.pronouns,input.target.languages?.length,input.target.badges?.length,input.target.representedGroup,input.target.statusDescription,input.target.dateJoined].filter(Boolean).length
  const graph = buildStrangerGraph(input)
  const coPresenceCount = graph.edges.filter(item=>item.kind==='co_presence').length
  const mutualIDs = new Set(input.mutuals.map(item=>item.id))
  const mutualsInGroups = new Set(input.members.filter(item=>mutualIDs.has(item.userId)).map(item=>item.userId))
  const mutualCoPresence = new Set(graph.edges.filter(item=>item.kind==='co_presence' && mutualIDs.has(item.target)).map(item=>item.target))
  const visibleMutualStatuses = input.mutuals.filter(item=>item.status && item.status!=='offline')
  const memberCounts = new Map<string,number>()
  for (const member of input.members) if (member.groupId) memberCounts.set(member.groupId,(memberCounts.get(member.groupId)||0)+1)
  const largestGroup = [...memberCounts.entries()].sort((a,b)=>b[1]-a[1])[0]
  const insights:RelationshipInsight[] = []
  if (mutualCount) insights.push({ id:'mutual-route', title:'共同好友是最直接的关系路径', summary:`当前可确认 ${mutualCount} 位共同好友；这表示你们之间存在可验证的社交桥梁，但不代表双方实际熟悉。`, confidence:mutualCount>=3?'高':'中', evidence:[`共同好友接口返回 ${mutualCount} 位`, ...input.mutuals.slice(0,3).map(item=>item.displayName)], tone:'strong' })
  insights.push({ id:'friend-slice', title:mutualCount?'已读取对方好友关系中可验证的交集':'未取得可验证的好友交集', summary:mutualCount?`当前账号能够确认 ${mutualCount} 位“既是你的好友、也是对方好友”的人物。VRChat 不向当前账号返回对方的完整好友列表，因此其余好友不能列出或估算。`:'共同好友接口没有返回明细；这可能是确实没有交集、隐私限制或接口不可用，不能解释为对方没有好友。', confidence:'高', evidence:[`共同好友计数：${mutualCount}`,`共同好友明细：${input.mutuals.length} 位`,'完整好友列表：接口未提供'], tone:'caution' })
  if (mutualsInGroups.size) insights.push({ id:'mutual-cross-source', title:'共同好友获得第二来源交叉印证', summary:`${mutualsInGroups.size} 位共同好友还出现在目标的可见群组成员中。好友交集与群组成员是两类独立接口，这些人物是更稳定的关系桥梁。`, confidence:'高', evidence:[...input.mutuals.filter(item=>mutualsInGroups.has(item.id)).slice(0,4).map(item=>`${item.displayName}：共同好友 + 可见群组成员`)], tone:'strong' })
  if (mutualCoPresence.size) insights.push({ id:'mutual-co-presence', title:'共同好友与目标存在同实例观察', summary:`本机日志发现 ${mutualCoPresence.size} 位共同好友曾与目标在同一实例、30 分钟窗口内出现。它比单纯群组重叠更接近真实活动证据，但仍不能证明双方交谈。`, confidence:'中', evidence:[...input.mutuals.filter(item=>mutualCoPresence.has(item.id)).slice(0,4).map(item=>`${item.displayName}：共同好友 + 同实例时间窗口`)], tone:'strong' })
  if (visibleMutualStatuses.length) insights.push({ id:'mutual-activity', title:'部分共同好友当前状态可见', summary:`共同好友明细中有 ${visibleMutualStatuses.length} 位显示非离线状态，可作为当前可联系桥梁参考；状态随时变化，不代表正在与目标互动。`, confidence:'高', evidence:visibleMutualStatuses.slice(0,4).map(item=>`${item.displayName}：${item.status}`), tone:'balanced' })
  if (groupCount) insights.push({ id:'group-overlap', title:'公开圈层存在重叠', summary:`目标公开了 ${input.groups.length} 个当前账号可见群组，接口显示共同群组计数 ${groupCount}。适合从群组活动和公开成员交集继续观察。`, confidence:input.groups.length>=2?'高':'中', evidence:input.groups.slice(0,4).map(item=>item.name), tone:'balanced' })
  if (bridge && bridge[1] > 1) {
    const person = input.members.find(item=>item.userId===bridge[0])
    insights.push({ id:'circle-bridge', title:'发现跨圈层桥接人物', summary:`${person?.displayName || bridge[0]} 同时出现在 ${bridge[1]} 个可见群组中，是当前公开关系图里的高连接候选。`, confidence:'中', evidence:[`${bridge[1]} 个公开群组重复出现`], tone:'strong' })
  }
  if (targetEvents.length) insights.push({ id:'local-observation', title:'存在本机活动证据', summary:`本机日志记录到该用户 ${targetEvents.length} 条事件。它只能证明你的客户端观察到相关活动，不能还原完整上线轨迹。`, confidence:'高', evidence:targetEvents.slice(0,3).map(item=>`${item.summary} · ${new Date(item.observedAt).toLocaleString('zh-CN')}`), tone:'balanced' })
  const structure = mutualCount>0&&groupCount>0?'混合桥接型':mutualCount>0?'共同好友主导型':groupCount>0?'公开群组主导型':'弱连接型'
  insights.push({ id:'network-shape', title:`关系结构偏向“${structure}”`, summary:structure==='混合桥接型'?'共同好友与公开群组两类路径同时存在，关系图不依赖单一来源。':structure==='共同好友主导型'?'当前主要证据来自共同好友，公开圈层信息较少。':structure==='公开群组主导型'?'当前主要证据来自公开群组，尚无共同好友明细支持。':'当前缺少稳定桥接节点，关系图主要用于记录后续证据。', confidence:(mutualCount||groupCount)?'中':'低', evidence:[`${mutualCount} 位共同好友`,`${groupCount} 个共同群组`,`${targetEvents.length} 条本机事件`], tone:(mutualCount&&groupCount)?'strong':'balanced' })
  if (largestGroup) {
    const group = input.groups.find(item=>item.id===largestGroup[0])
    const total = Math.max(1,input.members.length)
    const ratio = Math.round(largestGroup[1]/total*100)
    insights.push({ id:'circle-concentration', title:ratio>=60?'可见关系集中在单一圈层':'可见关系分布在多个圈层', summary:`${group?.name||largestGroup[0]} 占当前可见群组成员记录的 ${ratio}%。这反映本次采样的集中度，不代表目标完整社交结构。`, confidence:'中', evidence:[`${group?.name||largestGroup[0]}：${largestGroup[1]} 条成员记录`,`总样本：${input.members.length} 条`], tone:'balanced' })
  }
  if (coPresenceCount) insights.push({ id:'co-presence', title:'发现重复同实例线索', summary:`关系图发现 ${coPresenceCount} 条目标与候选人在同一实例、30 分钟窗口内出现的日志连线。它支持“曾被同时观察”，不能证明双方发生互动。`, confidence:'中', evidence:[`${coPresenceCount} 条同实例时间窗口匹配`,'数据来自本机 VRChat 日志'], tone:'strong' })
  if (input.target.dateJoined) {
    const joined = new Date(input.target.dateJoined)
    const years = Number.isNaN(joined.getTime())?0:Math.max(0,Math.floor((Date.now()-joined.getTime())/(365.25*24*60*60*1000)))
    insights.push({ id:'account-maturity', title:'账号历史与信任信息可交叉参考', summary:`账号约有 ${years} 年历史，当前信任显示为 ${input.target.trustLevel}。这只能描述账号公开元数据，不能直接代表个人可信度。`, confidence:'高', evidence:[`加入日期：${input.target.dateJoined}`,`信任字段：${input.target.trustLevel}`], tone:'balanced' })
  }
  const visibilitySignals = Number(Boolean(input.target.bio))+Number(Boolean(input.target.languages?.length))+Number(Boolean(input.groups.length))+Number(input.target.activityVisibility==='visible')
  insights.push({ id:'social-visibility', title:visibilitySignals>=3?'对当前账号公开的信息较丰富':'对当前账号公开的信息有限', summary:`4 类可见度信号中读取到 ${visibilitySignals} 类，包括简介、语言、群组和活动可见性。可见度受隐私设置和双方关系影响。`, confidence:'中', evidence:[`简介：${input.target.bio?'有':'无'}`,`语言：${input.target.languages?.length?'有':'无'}`,`群组：${input.groups.length}`,`活动：${input.target.activityVisibility||'未知'}`], tone:visibilitySignals>=3?'strong':'caution' })
  insights.push({ id:'profile-openness', title:profileFields>=5?'公开资料较完整':'公开资料较克制', summary:`当前读取到 ${profileFields}/7 类核心公开资料，活动信息${input.target.activityVisibility==='restricted'?'受限':'对当前账号可见'}。`, confidence:'中', evidence:[`资料来源：${input.target.profileSources?.join('、') || '基础用户接口'}`, `公开语言：${input.target.languages?.join('、') || '无'}`], tone:'balanced' })
  const routes = Number(mutualCount>0)+Number(groupCount>0)+Number(targetEvents.length>0)
  insights.push({ id:'contact-route', title:routes>=2?'存在多条可解释的接触路径':'当前关系证据仍然有限', summary:routes>=2?'共同好友、公开群组或本机记录中至少有两类证据。建议通过公开群组或共同好友自然确认关系，不依据图谱直接判断亲疏。':'目前不足以判断目标与你的实际关系，继续积累公开群组或本机同房证据更可靠。', confidence:routes>=2?'中':'低', evidence:[`${routes}/3 类关系来源有数据`], tone:routes>=2?'strong':'caution' })
  insights.push({ id:'coverage', title:'图谱覆盖范围提示', summary:'关系网只覆盖共同好友、当前账号可见群组和本机日志；私密好友、隐藏群组及未被本机观察的活动不会出现在图中。', confidence:'高', evidence:['访问权限边界','节点数量与请求频率限制'], tone:'caution' })
  return insights
}
