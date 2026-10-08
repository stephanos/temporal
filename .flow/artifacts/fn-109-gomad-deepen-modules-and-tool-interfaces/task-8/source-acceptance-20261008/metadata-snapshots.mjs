import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const known=new Map();
for(const file of fs.readdirSync(out).filter(n=>n.endsWith('.json'))){const bytes=fs.readFileSync(path.join(out,file));known.set(hash(bytes),path.join(out,file))}
for(const [digest,file]of [...known]){
 if(['source-controls.json','caller-controls.json','classification-controls.json'].includes(path.basename(file))){
  const snapshot=path.join(out,path.basename(file,'.json')+'-'+digest+'.json');
  if(!fs.existsSync(snapshot))fs.writeFileSync(snapshot,fs.readFileSync(file),{flag:'wx'});
  if(hash(fs.readFileSync(snapshot))!==digest)throw Error('metadata snapshot identity mismatch');known.set(digest,snapshot);
 }
}
function reconstructCaller(base){
 const inputs=[],overlays=[],changes=[];
 for(const [name,module]of [['original',base.original],['current',base.current]]){
  const overlay=path.join(base.scratch,name+'-caller-overlay.json');overlays.push(overlay);
  const Replace=JSON.parse(fs.readFileSync(overlay)).Replace;
  const profileSources=new Set(Object.keys(JSON.parse(fs.readFileSync(base.overlays[name==='original'?0:1])).Replace));
  for(const [source,replacement]of Object.entries(Replace)){
   if(profileSources.has(source))continue;
   const original=fs.existsSync(source)?fs.readFileSync(source,'utf8'):'',text=fs.readFileSync(replacement,'utf8');
   inputs.push({path:replacement,sha256:hash(text)});
   changes.push({name,file:path.relative(module,source),source_sha256:hash(original),overlay_sha256:hash(text),original_lines:original.split('\n').length,overlay_lines:text.split('\n').length});
  }
 }
 return JSON.stringify({...base,overlays,inputs:[...base.inputs,...inputs],changes,production:false},null,2)+'\n';
}
const rows=[],unresolved=[];
for(const file of fs.readdirSync(out).filter(n=>n.endsWith('-receipt.json')).sort()){
 const receipt=JSON.parse(fs.readFileSync(path.join(out,file))),manifest=JSON.parse(fs.readFileSync(path.join(out,receipt.control_manifest)));
 for(const entry of manifest.filter(x=>['source-controls.json','caller-controls.json','classification-controls.json'].includes(path.basename(x.path)))){
  let snapshot=known.get(entry.sha256);
  if(!snapshot&&path.basename(entry.path)==='caller-controls.json'){
   for(const basePath of [...new Set([...known.values()].filter(p=>path.basename(p).startsWith('source-controls-')))]){
    try{const bytes=reconstructCaller(JSON.parse(fs.readFileSync(basePath)));if(hash(bytes)===entry.sha256){snapshot=path.join(out,'caller-controls-'+entry.sha256+'.json');fs.writeFileSync(snapshot,bytes,{flag:'wx'});known.set(entry.sha256,snapshot);break}}catch{}
   }
  }
  if(snapshot)rows.push({receipt:file,original_path:entry.path,sha256:entry.sha256,snapshot:path.basename(snapshot),snapshot_sha256:hash(fs.readFileSync(snapshot)),method:path.basename(snapshot).startsWith('caller-controls-')?'Exact reconstruction from retained at-run graph/overlay bytes; SHA must equal recorded at-run identity':'Retained byte-identical metadata'});
  else unresolved.push({receipt:file,entry,reason:'No byte-identical retained/reconstructable metadata snapshot'});
 }
}
fs.writeFileSync(path.join(out,'at-run-metadata-snapshots-bound.json'),JSON.stringify({rows,unresolved,limitations:'Historical unresolved metadata is not final qualification authority. No receipt or raw log was changed; reconstructed snapshots are accepted only when exact recorded SHA matches.'},null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({snapshots:rows.length,unresolved:unresolved.map(x=>x.receipt)}));
