import fs from 'node:fs'
import path from 'node:path'
const [webSocketUrl, outputDirectory] = process.argv.slice(2)
if (!webSocketUrl) throw new Error('usage: node inspect-second-batch.mjs <ws-url> [output-dir]')
const socket = new WebSocket(webSocketUrl); let id=0; const pending=new Map()
const command=(method,params={})=>new Promise((resolve,reject)=>{const requestId=++id;const timer=setTimeout(()=>{pending.delete(requestId);reject(new Error(`${method} timed out`))},25000);pending.set(requestId,{resolve:(value)=>{clearTimeout(timer);resolve(value)},reject});socket.send(JSON.stringify({id:requestId,method,params}))})
socket.onmessage=({data})=>{const message=JSON.parse(data);const task=pending.get(message.id);if(!task)return;pending.delete(message.id);if(message.error)task.reject(new Error(message.error.message));else task.resolve(message.result)}
await new Promise((resolve,reject)=>{socket.onopen=resolve;socket.onerror=reject}); await command('Runtime.enable'); await command('Page.enable'); await command('Page.reload',{ignoreCache:true})
const wait=(ms)=>new Promise((resolve)=>setTimeout(resolve,ms)); await wait(3000)
const evaluate=async(expression)=>{const result=await command('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(result.exceptionDetails)throw new Error(result.exceptionDetails.text);return result.result.value}
const clickNav=async(label)=>{const clicked=await evaluate(`(()=>{const item=[...document.querySelectorAll('.sidebar nav button')].find(x=>x.textContent?.trim().startsWith(${JSON.stringify(label)}));item?.click();return Boolean(item)})()`);await wait(2200);return clicked}
const capture=async(name)=>{if(!outputDirectory)return;fs.mkdirSync(outputDirectory,{recursive:true});const result=await command('Page.captureScreenshot',{format:'png',fromSurface:true,captureBeyondViewport:false});fs.writeFileSync(path.join(outputDirectory,`${name}.png`),Buffer.from(result.data,'base64'))}

await evaluate(`(async()=>{for(let i=0;i<30;i++){if(document.querySelector('.sidebar'))return true;await new Promise(r=>setTimeout(r,250))}return false})()`)
await clickNav('群组');
const groups=await evaluate(`(()=>({cards:document.querySelectorAll('.group-list button').length,selected:document.querySelector('.group-profile h3')?.textContent?.trim()||'',posts:document.querySelectorAll('.group-columns article').length,instances:document.querySelectorAll('.instance-row').length,empty:document.querySelector('.group-empty')?.textContent?.trim()||'',error:document.querySelector('.group-error')?.textContent?.trim()||'',images:[...document.querySelectorAll('.group-center img')].filter(x=>x.complete&&x.naturalWidth>0).length}))()`);await capture('second-batch-groups')
await clickNav('头像');
const avatars=await evaluate(`(()=>({cards:document.querySelectorAll('.avatar-grid>button').length,count:document.querySelector('.avatar-filters>span')?.textContent?.trim()||'',loadedImages:[...document.querySelectorAll('.avatar-grid img')].filter(x=>x.complete&&x.naturalWidth>0).length,error:document.querySelector('.avatar-error')?.textContent?.trim()||''}))()`);await capture('second-batch-avatars')
await evaluate(`document.querySelector('.avatar-grid>button')?.click()`);await wait(400)
const avatarDetail=await evaluate(`(()=>({open:Boolean(document.querySelector('.avatar-detail')),name:document.querySelector('.avatar-detail h3')?.textContent?.trim()||'',platformRows:document.querySelectorAll('.avatar-detail dl>div').length,imageLoaded:Boolean([...document.querySelectorAll('.avatar-layout>aside>img')].find(x=>x.complete&&x.naturalWidth>0))}))()`)
await clickNav('世界');
const worlds=await evaluate(`(()=>({memorySections:document.querySelectorAll('.memory-grid>section').length,memories:document.querySelectorAll('.memory-grid>section:first-child button').length,recommendations:document.querySelectorAll('.memory-grid>section:last-child button').length,worldCards:document.querySelectorAll('.world-card').length}))()`);await capture('second-batch-worlds')
await evaluate(`document.querySelector('.world-card')?.click()`);await wait(900);await evaluate(`document.querySelector('.world-drawer .detail-toolbar button, .world-detail .detail-toolbar button')?.click()`);await clickNav('总览')
const recent=await evaluate(`(()=>({items:document.querySelectorAll('.recent-access button').length,text:document.querySelector('.recent-access')?.textContent?.replace(/\s+/g,' ').trim()||''}))()`);await capture('second-batch-overview')
const page=await evaluate(`(()=>({errors:[...document.querySelectorAll('.error-banner')].map(x=>x.textContent?.trim()),bodyWidth:document.body.scrollWidth,viewport:window.innerWidth,title:document.title}))()`)
console.log(JSON.stringify({groups,avatars,avatarDetail,worlds,recent,page},null,2));socket.close()
