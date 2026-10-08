import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {sourceManifestBinding,sourceManifestIndex} from './source-manifests.mjs';

const out=path.dirname(new URL(import.meta.url).pathname),indexPath=path.join(out,'source-manifest-index.json'),bindingPath=path.join(out,'manifest-compaction.json');
const hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex');
const snapshot=name=>{const bytes=fs.readFileSync(path.join(out,name));return {path:name,sha256:hash(bytes),bytes:bytes.length};};
const protectedNames=()=>fs.readdirSync(out).filter(name=>name.endsWith('-receipt.json')||name.endsWith('.log')).sort().concat(['run.mjs','instrument-publication-executed.mjs','upgrade-publication-overlay-binding.json','green-packet-union.json','r4-named-assertions.json','original-owner-mapping.json','preservation.json','lint-attribution.json']);
const indexFile=path.resolve(process.cwd(),'.git/index');
const verifyProtected=binding=>{
 if(hash(fs.readFileSync(indexFile))!==binding.staged_index_sha256)throw Error('staged index changed');
 for(const row of binding.protected_artifacts){const actual=snapshot(row.path);if(actual.sha256!==row.sha256||actual.bytes!==row.bytes)throw Error('protected artifact changed '+row.path);}
};
const mode=process.argv[2];
if(mode==='--prepare'){
 if(fs.existsSync(indexPath)||fs.existsSync(bindingPath))throw Error('compaction already prepared');
 const originals=fs.readdirSync(out).filter(name=>name.endsWith('-source.json')).sort(),groups=new Map();
 for(const original of originals){const bytes=fs.readFileSync(path.join(out,original)),sha256=hash(bytes);if(!groups.has(sha256))groups.set(sha256,[]);groups.get(sha256).push({original,bytes,sha256});}
 if(originals.length!==32||groups.size!==9)throw Error('unexpected original inventory');
 const instrumenter=JSON.parse(fs.readFileSync(path.join(out,'upgrade-publication-overlay-binding.json')));
 if(snapshot('instrument-publication-executed.mjs').sha256!==instrumenter.instrumenter_sha256)throw Error('executed original instrumenter provenance differs');
 const preferred=['baseline-lint-source.json','final-cli-packs-source.json','diagnostics-checked-source.json','diagnostics-characterized-source.json'];
 const bindings=[];
 for(const group of groups.values()){
  const canonical=preferred.find(name=>group.some(row=>row.original===name))??group[0].original;
  const retained=group.find(row=>row.original===canonical).bytes;
  for(const row of group){if(!row.bytes.equals(retained))throw Error('hash grouping is not exact byte equality');bindings.push({gate:row.original.slice(0,-'-source.json'.length),original:row.original,canonical,original_payload_sha256:row.sha256,original_bytes:row.bytes.length,source_tree_sha256:hash(JSON.stringify(JSON.parse(row.bytes)))});}
 }
 bindings.sort((a,b)=>a.original.localeCompare(b.original));
 const originalBytes=bindings.reduce((sum,row)=>sum+row.original_bytes,0),canonicalBytes=[...groups.values()].reduce((sum,group)=>sum+group[0].bytes.length,0);
 const index={version:1,artifact_directory:out,logical_gate_bindings:32,unique_exact_byte_manifests:9,duplicate_files:23,original_bytes:originalBytes,retained_bytes:canonicalBytes,removed_redundant_bytes:originalBytes-canonicalBytes,recovery:'For each original filename, copy its canonical payload verbatim; original_payload_sha256 and original_bytes bind the exact original bytes, including formatting and EOF.',bindings};
 fs.writeFileSync(indexPath,JSON.stringify(index,null,2)+'\n');
 for(const row of bindings)sourceManifestBinding(row.original);
 const binding={scope:'task3 source-acceptance directory only; artifact-only lossless exact-byte deduplication',source_manifest_index:'source-manifest-index.json',index_sha256:hash(fs.readFileSync(indexPath)),staged_index_sha256:hash(fs.readFileSync(indexFile)),protected_artifacts:protectedNames().map(snapshot),raw_receipts_logs_and_harness_unchanged:true,packet_and_assertion_metadata_unchanged:true,source_tree_sha256:'4d67c77fcaa52379bebfc0a61658a34a3b57ffcfcd77f2a5964815eb5fb8ea10',logical_gate_bindings:32,canonical_manifests:9,duplicate_files:23,removed_redundant_bytes:index.removed_redundant_bytes,all_original_payloads_compared_byte_exact_before_deletion:true,deleted:[]};
 fs.writeFileSync(bindingPath,JSON.stringify(binding,null,2)+'\n');
 console.log(JSON.stringify({prepared:true,original_bytes:originalBytes,retained_bytes:canonicalBytes,logical_gate_bindings:32,canonical_manifests:9}));
}else if(mode==='--delete-duplicates'||mode==='--verify'){
 const binding=JSON.parse(fs.readFileSync(bindingPath)),index=sourceManifestIndex();
 if(hash(fs.readFileSync(indexPath))!==binding.index_sha256)throw Error('manifest index changed');
 verifyProtected(binding);
 for(const row of index.bindings){const actual=sourceManifestBinding(row.original);const receipt=JSON.parse(fs.readFileSync(path.join(out,row.gate+'-receipt.json')));if(actual.source_tree_sha256!==receipt.source_tree_sha256)throw Error('receipt source binding differs '+row.gate);}
 if(mode==='--delete-duplicates'){
  if(binding.deleted.length)throw Error('duplicate deletion already recorded');
  const duplicates=index.bindings.filter(row=>row.original!==row.canonical);
  for(const row of duplicates){const file=path.join(out,row.original);if(path.dirname(file)!==out||!fs.readFileSync(file).equals(fs.readFileSync(path.join(out,row.canonical))))throw Error('deletion target is not an exact owned duplicate');}
  for(const row of duplicates){fs.unlinkSync(path.join(out,row.original));binding.deleted.push(row.original);}
  fs.writeFileSync(bindingPath,JSON.stringify(binding,null,2)+'\n');
 }
 for(const row of index.bindings)sourceManifestBinding(row.original);
 verifyProtected(binding);
 const retained=fs.readdirSync(out).filter(name=>name.endsWith('-source.json'));
 if(binding.deleted.length!==23||retained.length!==9||index.bindings.length!==32)throw Error('incomplete compaction');
 console.log(JSON.stringify({verified:true,logical_gate_bindings:32,canonical_manifests:9,deleted_exact_duplicates:23,recoverable_byte_exact:true,protected_artifacts:binding.protected_artifacts.length,staged_index_unchanged:true,final_source_tree_sha256:binding.source_tree_sha256}));
}else throw Error('expected --prepare, --delete-duplicates or --verify');
