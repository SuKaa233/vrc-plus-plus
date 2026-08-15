import { describe, expect, it } from 'vitest'
import { buildReunionCards, buildTonightCards, queryLocalSocial } from './insights'
import type { ActivityEvent, Friend, World } from './api'

const friends: Friend[] = [
  {id:'usr_a',displayName:'Alpha',online:true,location:'wrld_x:1'},
  {id:'usr_b',displayName:'Beta',online:true,location:'wrld_x:1'},
  {id:'usr_c',displayName:'Gamma',online:true},
]
const worlds: World[]=[{id:'wrld_x',name:'Night Cafe'}]
const events: ActivityEvent[]=[
  {id:'1',type:'game.player-joined',userId:'usr_c',worldId:'wrld_x',summary:'join',observedAt:'2026-07-01T12:00:00Z'},
  {id:'2',type:'game.player-left',userId:'usr_c',worldId:'wrld_x',summary:'left',observedAt:'2026-07-01T13:00:00Z'},
  {id:'3',type:'game.player-joined',userId:'usr_a',summary:'join',observedAt:'2026-08-14T12:00:00Z'},
]

describe('local insights',()=>{
  it('ranks friend clusters for tonight',()=>{const cards=buildTonightCards(friends,worlds,['wrld_x'],events);expect(cards).toHaveLength(1);expect(cards[0].title).toBe('Night Cafe');expect(cards[0].score).toBeGreaterThan(60)})
  it('finds evidence-backed reunions',()=>{const cards=buildReunionCards(friends,events,new Date('2026-08-15T12:00:00Z'));expect(cards[0].targetUserId).toBe('usr_c')})
  it('returns a coverage answer without guessing',()=>{const result=queryLocalSocial('数据覆盖怎么样',friends,worlds,events,null);expect(result.intent).toBe('coverage');expect(result.cards[0].coverageDays).toBe(2)})
  it('returns help for unsupported questions',()=>{expect(queryLocalSocial('随便问问',friends,worlds,events,null).intent).toBe('help')})
})
