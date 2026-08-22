import { layoutFriendNetwork } from './friend-network'
import type { FriendNetworkEdge, FriendNetworkNode } from './api'

self.onmessage = (event: MessageEvent<{ id:number; nodes:FriendNetworkNode[]; edges:FriendNetworkEdge[]; width:number; height:number }>) => {
  const { id,nodes,edges,width,height }=event.data
  self.postMessage({id,positions:layoutFriendNetwork(nodes,edges,width,height)})
}
