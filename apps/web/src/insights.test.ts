import { describe, expect, it } from 'vitest'
import { buildReunionCards, buildTonightCards, queryLocalSocial } from './insights'
import type { ActivityEvent, Friend, World } from './api'

const friends: Friend[] = [
  {id:'usr_a',displayName:'Alpha',online:true,location:'wrld_x:1'},
  {id:'usr_b',displayName:'Beta',online:true,location:'wrld_x:1'},
  {id:'usr_c',displayName:'Gamma',online:true},
]
const worlds: World[]=[{id:'wrld_x',name:'Night Cafe'},{id:'wrld_y',name:'Quiet Garden',visits:12000}]
const events: ActivityEvent[]=[
  {id:'1',type:'game.player-joined',userId:'usr_c',worldId:'wrld_x',summary:'join',observedAt:'2026-07-01T12:00:00Z'},
  {id:'2',type:'game.player-left',userId:'usr_c',worldId:'wrld_x',summary:'left',observedAt:'2026-07-01T13:00:00Z'},
  {id:'3',type:'game.player-joined',userId:'usr_a',summary:'join',observedAt:'2026-08-14T12:00:00Z'},
]

describe('local insights',()=>{
  it('ranks friend clusters before fallback recommendations',()=>{const cards=buildTonightCards(friends,worlds,['wrld_x'],events);expect(cards).toHaveLength(2);expect(cards[0].title).toBe('Night Cafe');expect(cards[0].location).toBe('wrld_x:1');expect(cards[0].score).toBeGreaterThan(60)})
  it('falls back to evidence-labelled worlds when no friend location is joinable',()=>{const cards=buildTonightCards([{...friends[2],online:false}],worlds,['wrld_y'],events);expect(cards.length).toBeGreaterThan(0);expect(cards[0].location).toBeUndefined();expect(cards.some(item=>item.title==='Quiet Garden')).toBe(true)})
  it('finds evidence-backed reunions',()=>{const cards=buildReunionCards(friends,events,new Date('2026-08-15T12:00:00Z'));expect(cards[0].targetUserId).toBe('usr_c')})
  it('returns a coverage answer without guessing',()=>{const result=queryLocalSocial('数据覆盖怎么样',friends,worlds,events,null);expect(result.intent).toBe('coverage');expect(result.cards[0].coverageDays).toBe(2)})
  it('returns help for unsupported questions',()=>{expect(queryLocalSocial('随便问问',friends,worlds,events,null).intent).toBe('help')})
  it('answers tonight recommendations without requiring a public friend location',()=>{const result=queryLocalSocial('今晚去哪',[{...friends[2],online:false}],worlds,events,null);expect(result.intent).toBe('tonight');expect(result.cards.length).toBeGreaterThan(0)})
  it('supports distinct compass strategies without inventing friend locations',()=>{const explore=buildTonightCards(friends,worlds,['wrld_y'],events,'explore');expect(explore[0].targetWorldId).toBe('wrld_y');expect(explore.filter(item=>!item.location).every(item=>item.reasons.some(reason=>reason.includes('不代表有好友')))).toBe(true)})
})
