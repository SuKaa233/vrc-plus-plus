import type { ActivityEvent, FriendActivityInsights } from './api'

export interface ConcernedInference {
  id: string
  title: string
  summary: string
  evidence: string
  confidence: '较高' | '中等' | '较低'
  tone: 'accent' | 'success' | 'warning' | 'neutral'
}

export interface ConcernedInferenceInput {
  insights: FriendActivityInsights | null
  events: ActivityEvent[]
  relationPeerCount: number
  allNetworkDegrees: number[]
  coObservedCount: number
  coObservedRelationOverlap: number
  visitedWorlds: Array<{ count: number; privateCount: number }>
}

export function buildConcernedInferences(input: ConcernedInferenceInput): ConcernedInference[] {
  const result: ConcernedInference[] = []
  const insights = input.insights

  if (input.allNetworkDegrees.length && input.relationPeerCount > 0) {
    const percentile = Math.round(input.allNetworkDegrees.filter((degree) => degree <= input.relationPeerCount).length / input.allNetworkDegrees.length * 100)
    const connector = percentile >= 75
    result.push({
      id: 'network-role',
      title: connector ? '可能是关系网中的连接型好友' : '当前关系连接更偏局部',
      summary: connector ? '已观察连接数位于当前关系网较高区间，可能连接多个你可见的朋友圈。' : '已观察连接主要集中在较小范围，当前证据更像局部圈层。',
      evidence: `${input.relationPeerCount} 条当前连线 · 约高于 ${percentile}% 的已载入节点`,
      confidence: input.allNetworkDegrees.length >= 30 ? '中等' : '较低',
      tone: connector ? 'accent' : 'neutral',
    })
  }

  const changes = insights?.relationChanges ?? []
  const added = changes.filter((item) => item.state === 'newly_observed').length
  const missing = changes.filter((item) => item.state === 'not_observed').length
  if (added || missing) {
    result.push({
      id: 'relation-trend',
      title: added > missing ? '近期可见关系范围可能在扩展' : added < missing ? '近期部分关系未再次观察到' : '可见关系近期有双向变化',
      summary: '这里只比较共同好友扫描结果，不等同于确认其真实新增或删除好友。',
      evidence: `新观察 ${added} 条 · 未再次观察 ${missing} 条`,
      confidence: changes.length >= 4 ? '中等' : '较低',
      tone: added > missing ? 'success' : missing > added ? 'warning' : 'neutral',
    })
  }

  const locationKinds = insights?.locationKinds ?? {}
  const visibleLocations = ['private', 'friends_plus', 'friends', 'invite_plus', 'group', 'public']
    .reduce((total, key) => total + (locationKinds[key] ?? 0), 0)
  if (visibleLocations > 0) {
    const privateCount = locationKinds.private ?? 0
    const privateRatio = Math.round(privateCount / visibleLocations * 100)
    const socialCount = (locationKinds.friends_plus ?? 0) + (locationKinds.friends ?? 0) + (locationKinds.invite_plus ?? 0) + (locationKinds.group ?? 0)
    result.push({
      id: 'instance-style',
      title: privateRatio >= 55 ? '可见轨迹更偏私人空间' : socialCount > privateCount ? '可见轨迹更偏社交型实例' : '实例类型分布较均衡',
      summary: privateRatio >= 55 ? '本机观察到的可见位置中，私人实例占比较高。' : '好友、邀请或群组实例在可见轨迹中占比较高。',
      evidence: `可分类位置 ${visibleLocations} 条 · 私人位置 ${privateRatio}%`,
      confidence: visibleLocations >= 12 ? '较高' : visibleLocations >= 5 ? '中等' : '较低',
      tone: privateRatio >= 55 ? 'warning' : 'accent',
    })
  }

  if (input.visitedWorlds.length) {
    const observations = input.visitedWorlds.reduce((total, item) => total + item.count, 0)
    const repeatRate = Math.max(0, Math.round((observations - input.visitedWorlds.length) / Math.max(1, observations) * 100))
    result.push({
      id: 'world-style',
      title: repeatRate >= 50 ? '可能偏好重复回访熟悉世界' : '可见世界轨迹更偏探索',
      summary: repeatRate >= 50 ? '同一批世界在本机记录中重复出现较多。' : '可见记录分散在多个世界，重复度暂时不高。',
      evidence: `${input.visitedWorlds.length} 个世界 · ${observations} 条世界记录 · 重复度 ${repeatRate}%`,
      confidence: observations >= 10 ? '中等' : '较低',
      tone: 'neutral',
    })
  }

  const activeHours = insights?.activeHours ?? []
  if (activeHours.length) {
    const top = activeHours.slice(0, 3).map((item) => `${String(item.hour).padStart(2, '0')}:00`).join('、')
    const evidenceCount = activeHours.reduce((total, item) => total + item.count, 0)
    result.push({
      id: 'active-rhythm',
      title: '本机记录显示出常见活跃时段',
      summary: `当前最常被观察到的时段集中在 ${top} 附近，可作为寻找对方的时间参考。`,
      evidence: `${evidenceCount} 条时段样本 · 覆盖 ${insights?.coverageDays ?? 0} 天`,
      confidence: (insights?.coverageDays ?? 0) >= 7 ? '中等' : '较低',
      tone: 'accent',
    })
  }

  if (input.coObservedCount > 0) {
    const overlapRatio = Math.round(input.coObservedRelationOverlap / input.coObservedCount * 100)
    result.push({
      id: 'co-presence',
      title: overlapRatio >= 50 ? '同场线索与关系圈重合较多' : '同场线索包含较多圈外人物',
      summary: overlapRatio >= 50 ? '邻近日志里出现的人多数也在当前关系连线中，可能存在相对稳定的可见活动圈。' : '邻近日志中有不少人不在当前关系连线里，可能来自临时同场或尚未扫描完整的关系。',
      evidence: `${input.coObservedCount} 位同场时段人物 · ${input.coObservedRelationOverlap} 位同时有关系线索`,
      confidence: input.coObservedCount >= 6 ? '中等' : '较低',
      tone: overlapRatio >= 50 ? 'success' : 'neutral',
    })
  }

  const recentBoundary = Date.now() - 7 * 24 * 60 * 60 * 1000
  const recentEvents = input.events.filter((event) => new Date(event.observedAt).getTime() >= recentBoundary).length
  if (input.events.length >= 4) {
    result.push({
      id: 'recent-activity',
      title: recentEvents >= Math.ceil(input.events.length * .6) ? '最近一周观察活动较集中' : '记录分布不只集中在最近一周',
      summary: '这反映的是应用运行并成功接收事件时的覆盖，不代表对方完整上线频率。',
      evidence: `最近 7 天 ${recentEvents} 条 · 当前档案共 ${input.events.length} 条`,
      confidence: (insights?.coverageDays ?? 0) >= 5 ? '中等' : '较低',
      tone: 'neutral',
    })
  }

  return result
}
