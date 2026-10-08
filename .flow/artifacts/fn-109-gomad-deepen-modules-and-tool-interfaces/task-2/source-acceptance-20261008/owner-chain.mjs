import assert from 'node:assert/strict';
import {writeFileSync,existsSync} from 'node:fs';
import {root,out,original,sha,read,git,reconstruct} from './reconstruct.mjs';
export function apply(patch,before,path){
 const lines=before?before.split('\n').slice(0,-1):[],result=[];let cursor=0;
 const section=patch.split('\n');
 for(let n=0;n<section.length;n++){
  const m=/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(section[n]);if(!m)continue;
  const oc=Number(m[2]??1),nc=Number(m[4]??1),offset=oc?Number(m[1])-1:Number(m[1]);assert(offset>=cursor,path);result.push(...lines.slice(cursor,offset));cursor=offset;let o=0,k=0;
  while(o<oc||k<nc){const s=section[++n],c=s[0],t=s.slice(1);assert([' ','+','-'].includes(c));if(c!=='+'){assert.equal(lines[cursor++],t,path);o++;}if(c!=='-'){result.push(t);k++;}}
 }
 result.push(...lines.slice(cursor));return result.join('\n')+'\n';
}
export function patchImages(task){
 const dir=original.replace(/task-2$/,`task-${task}`),patch=read(dir+'/task-only.patch').toString();
 const headers=[...patch.matchAll(/^--- (a\/[^\n]+|\/dev\/null)\n\+\+\+ b\/([^\n]+)\n/gm)],images={},pre={};
 const final=JSON.parse(read(dir+'/final-source.json'));
 for(let i=0;i<headers.length;i++){
  const h=headers[i],path=h[2];let b='';const ent=task===5?final.files[path]:final.files.find(e=>e.path===path);assert(ent,path);
  if(h[1]!=='/dev/null'){b=read(dir+(task===5?'/before/':'/preimages/')+path).toString();assert.equal(sha(b),task===5?ent.before_sha256:ent.pre_sha256,path+' owner preimage');}
  pre[path]=b;images[path]=apply(patch.slice(h.index+h[0].length,i+1<headers.length?headers[i+1].index:patch.length),b,path);
  assert.equal(sha(images[path]),task===5?ent.after_sha256:ent.post_sha256,path+' owner postimage');
 }
 return {images,pre,patch_sha256:sha(patch),paths:headers.length};
}
export function chain(){
 const r=reconstruct(),five=patchImages(5),six=patchImages(6);
 const rows=r.rows.map(row=>{
  const path=row.path,history=git(['log','--first-parent','--format=%H %s','a3b9f80efa..HEAD','--',path]).toString().trim().split('\n').filter(Boolean);
  const p5=five.images[path],p6=six.images[path],a3=git(['show','a3b9f80efa:'+path]);
  return {...row,task5:p5?{pre_sha256:sha(five.pre[path]),post_sha256:sha(p5),directly_from_task2:sha(five.pre[path])===row.postimage_sha256}:null,task6:p6?{pre_sha256:sha(six.pre[path]),post_sha256:sha(p6),from_task5:p5?sha(six.pre[path])===sha(p5):null,directly_from_task2:sha(six.pre[path])===row.postimage_sha256,equals_current:sha(p6)===row.current_sha256}:null,integrated_a3_sha256:sha(a3),integrated_a3_equals_task6:p6?sha(a3)===sha(p6):null,current_equals_integrated:sha(a3)===row.current_sha256,later_first_parent_commits:history};
 });
 return {task5_patch_sha256:five.patch_sha256,task5_verified_paths:five.paths,task6_patch_sha256:six.patch_sha256,task6_verified_paths:six.paths,rows};
}
if(process.argv[1]===new URL(import.meta.url).pathname){assert(!existsSync(out+'/owner-chain.json'));const r=chain();writeFileSync(out+'/owner-chain.json',JSON.stringify(r,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify(r.rows.map(e=>({path:e.path,task6_current:e.task6?.equals_current,a3_current:e.current_equals_integrated,commits:e.later_first_parent_commits}))));}
