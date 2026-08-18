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
    const matches = (eventByUser.get(candidate.userId) || []).some(event => targetEvents.some(target => {
      const samePlace = event.location && target.location && event.location === target.location
      const close = Math.abs(new Date(event.observedAt).getTime() - new Date(target.observedAt).getTime()) <= 30*60*1000
      return samePlace && close
    }))
    if (matches) edges.push({ source:input.target.id, target:candidate.userId, kind:'co_presence', label:'日志同实例', evidence:'observed' })
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
  const insights:RelationshipInsight[] = []
  if (mutualCount) insights.push({ id:'mutual-route', title:'共同好友是最直接的关系路径', summary:`当前可确认 ${mutualCount} 位共同好友；这表示你们之间存在可验证的社交桥梁，但不代表双方实际熟悉。`, confidence:mutualCount>=3?'高':'中', evidence:[`共同好友接口返回 ${mutualCount} 位`, ...input.mutuals.slice(0,3).map(item=>item.displayName)], tone:'strong' })
  if (groupCount) insights.push({ id:'group-overlap', title:'公开圈层存在重叠', summary:`目标公开了 ${input.groups.length} 个当前账号可见群组，接口显示共同群组计数 ${groupCount}。适合从群组活动和公开成员交集继续观察。`, confidence:input.groups.length>=2?'高':'中', evidence:input.groups.slice(0,4).map(item=>item.name), tone:'balanced' })
  if (bridge && bridge[1] > 1) {
    const person = input.members.find(item=>item.userId===bridge[0])
    insights.push({ id:'circle-bridge', title:'发现跨圈层桥接人物', summary:`${person?.displayName || bridge[0]} 同时出现在 ${bridge[1]} 个可见群组中，是当前公开关系图里的高连接候选。`, confidence:'中', evidence:[`${bridge[1]} 个公开群组重复出现`], tone:'strong' })
  }
  if (targetEvents.length) insights.push({ id:'local-observation', title:'存在本机活动证据', summary:`本机日志记录到该用户 ${targetEvents.length} 条事件。它只能证明你的客户端观察到相关活动，不能还原完整上线轨迹。`, confidence:'高', evidence:targetEvents.slice(0,3).map(item=>`${item.summary} · ${new Date(item.observedAt).toLocaleString('zh-CN')}`), tone:'balanced' })
  insights.push({ id:'profile-openness', title:profileFields>=5?'公开资料较完整':'公开资料较克制', summary:`当前读取到 ${profileFields}/7 类核心公开资料，活动信息${input.target.activityVisibility==='restricted'?'受限':'对当前账号可见'}。`, confidence:'中', evidence:[`资料来源：${input.target.profileSources?.join('、') || '基础用户接口'}`, `公开语言：${input.target.languages?.join('、') || '无'}`], tone:'balanced' })
  const routes = Number(mutualCount>0)+Number(groupCount>0)+Number(targetEvents.length>0)
  insights.push({ id:'contact-route', title:routes>=2?'存在多条可解释的接触路径':'当前关系证据仍然有限', summary:routes>=2?'共同好友、公开群组或本机记录中至少有两类证据。建议通过公开群组或共同好友自然确认关系，不依据图谱直接判断亲疏。':'目前不足以判断目标与你的实际关系，继续积累公开群组或本机同房证据更可靠。', confidence:routes>=2?'中':'低', evidence:[`${routes}/3 类关系来源有数据`], tone:routes>=2?'strong':'caution' })
  insights.push({ id:'coverage', title:'图谱覆盖范围提示', summary:'关系网只覆盖共同好友、当前账号可见群组和本机日志；私密好友、隐藏群组及未被本机观察的活动不会出现在图中。', confidence:'高', evidence:['访问权限边界','节点数量与请求频率限制'], tone:'caution' })
  return insights
}
