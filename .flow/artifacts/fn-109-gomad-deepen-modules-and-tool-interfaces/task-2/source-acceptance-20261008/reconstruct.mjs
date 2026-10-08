import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {readFileSync,writeFileSync,existsSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
export const root='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=dirname(fileURLToPath(import.meta.url));
export const original='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-2';
export const sha=b=>createHash('sha256').update(b).digest('hex');
export const read=p=>readFileSync(resolve(root,p));
export function git(argv){const r=spawnSync('git',argv,{cwd:root,maxBuffer:32<<20});assert.equal(r.status,0,r.stderr.toString());return r.stdout;}
export function reconstruct(){
 const pre=JSON.parse(read(original+'/source-pre-complete.json')),post=JSON.parse(read(original+'/source-post-complete.json'));
 const patch=read(original+'/own-work.patch').toString(),headers=[...patch.matchAll(/^--- (a\/[^\n]+|\/dev\/null)\n\+\+\+ b\/([^\n]+)\n/gm)],images={},preimages={},rows=[];
 assert.equal(headers.length,20);
 for(let i=0;i<headers.length;i++){
  const h=headers[i],path=h[2],start=h.index+h[0].length,end=i+1<headers.length?headers[i+1].index:patch.length;
  const beforeHeader=patch.slice(i?headers[i-1].index:0,h.index),index=/index ([a-f0-9]+)\.\.([a-f0-9]+)/g;let m,last;while((m=index.exec(beforeHeader)))last=m;
  const entry=pre.find(e=>e.path===path);assert(entry);
  let body=Buffer.alloc(0),origin='new file';
  if(entry.sha256){
   if(entry.source==='parent-verified pre-task image'){origin=original+'/parent-verified-preimages/'+path.split('/').at(-1);body=read(origin);}
   else{assert(last,path);origin='git blob '+last[1];body=git(['cat-file','blob',last[1]]);}
   assert.equal(sha(body),entry.sha256,path+' preimage');
  }
  preimages[path]=body.toString();
  const lines=body.length?body.toString().split('\n').slice(0,-1):[],section=patch.slice(start,end).split('\n'),result=[];let cursor=0,hunks=0;
  for(let n=0;n<section.length;n++){
   const match=/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(section[n]);if(!match)continue;
   hunks++;const oldStart=Number(match[1]),oldCount=Number(match[2]??1),newCount=Number(match[4]??1),offset=oldCount?oldStart-1:oldStart;
   assert(offset>=cursor,path+' hunk order');result.push(...lines.slice(cursor,offset));cursor=offset;
   let oldSeen=0,newSeen=0;
   while(oldSeen<oldCount||newSeen<newCount){const line=section[++n],kind=line[0],text=line.slice(1);assert([' ','+','-'].includes(kind),path+' hunk line');if(kind!=='+' ){assert.equal(lines[cursor++],text,path+' exact context');oldSeen++;}if(kind!=='-'){result.push(text);newSeen++;}}
   assert.equal(oldSeen,oldCount);assert.equal(newSeen,newCount);
  }
  result.push(...lines.slice(cursor));const after=result.join('\n')+'\n';assert.equal(sha(after),post.find(e=>e.path===path).sha256,path+' postimage');images[path]=after;
  rows.push({path,preimage_sha256:entry.sha256,postimage_sha256:sha(after),origin,hunks,current_sha256:sha(read(path)),current_equals_original:sha(read(path))===sha(after)});
 }
 return {rows,images,preimages};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){
 assert(!existsSync(out+'/historical-reconstruction.json'));
 const r=reconstruct();writeFileSync(out+'/historical-reconstruction.json',JSON.stringify({patch_sha256:sha(read(original+'/own-work.patch')),method:'Exact Git blob or retained parent preimages; each hunk consumed at its original line with byte-equal context; all20 pre/post SHA-256 verified in memory, no later tree substituted',rows:r.rows},null,2)+'\n',{flag:'wx'});
 console.log(JSON.stringify({postimages:r.rows.length,unchanged:r.rows.filter(e=>e.current_equals_original).length,changed:r.rows.filter(e=>!e.current_equals_original).length}));
}
