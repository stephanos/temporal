import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {read,sha,git,out} from './capture.mjs';
import {apply,patchImages} from '../../task-2/source-acceptance-20261008/owner-chain.mjs';
export const original='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-3';
export function reconstruct(){
 const pre=JSON.parse(read(original+'/source-pre.json')),post=JSON.parse(read(original+'/source-post.json')),patch=read(original+'/task-only.patch').toString(),heads=[...patch.matchAll(/^--- (a\/[^\n]+|\/dev\/null)\n\+\+\+ b\/([^\n]+)\n/gm)],images={},preimages={},rows=[];
 assert.equal(heads.length,5);
 for(let i=0;i<heads.length;i++){
  const h=heads[i],path=h[2],before=h[1]==='/dev/null'?'':read(original+'/preimages/'+path).toString();
  if(before)assert.equal(sha(before),pre[path],path+' original preimage');
  const section=patch.slice(h.index+h[0].length,i+1<heads.length?heads[i+1].index:patch.length),after=apply(section,before,path);assert.equal(sha(after),post[path],path+' original postimage');
  images[path]=after;preimages[path]=before;rows.push({path,pre_sha256:before?sha(before):null,post_sha256:sha(after),current_sha256:sha(read(path)),unchanged:sha(after)===sha(read(path)),hunks:[...section.matchAll(/^@@.*$/gm)].map(m=>m[0])});
 }
 return {preimages,images,rows};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){const r=reconstruct();writeFileSync(out+'/historical-reconstruction.json',JSON.stringify({patch_sha256:sha(read(original+'/task-only.patch')),four_modified_images:4,new_characterization_images:1,rows:r.rows},null,2)+'\n',{flag:'wx'});console.log(JSON.stringify(r.rows));}
