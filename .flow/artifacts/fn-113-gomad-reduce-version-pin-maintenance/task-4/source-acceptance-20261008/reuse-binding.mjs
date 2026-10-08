import fs from 'node:fs';
import path from 'node:path';
import {repo,out,hash,git} from './run.mjs';
const predecessor=path.resolve(out,'../../task-2/source-acceptance-20261008');
const manifestPath=path.join(predecessor,'accepted-deterministicio-workspace-complement-source.json');
const oldManifest=new Map(JSON.parse(fs.readFileSync(manifestPath)).map(x=>[x.path,x.sha256]));
const source=fs.readFileSync(path.join(out,'reuse-deterministicio-dependencies.log'),'utf8');
const objects=[];let start=-1,depth=0,quoted=false,escape=false;
for(let i=0;i<source.length;i++){const c=source[i];if(quoted){if(escape)escape=false;else if(c==='\\')escape=true;else if(c==='"')quoted=false;}else if(c==='"')quoted=true;else if(c==='{'){if(depth===0)start=i;depth++;}else if(c==='}'){depth--;if(depth===0)objects.push(JSON.parse(source.slice(start,i+1)));}}
const paths=new Set(['go.mod','go.sum','tools/gomad3/go.mod','tools/gomad3/go.sum']),generated=[];
for(const p of objects){if(!p.Dir?.startsWith(path.join(repo,'tools/gomad3')))continue;for(const f of [...p.GoFiles??[],...p.EmbedFiles??[],...p.CgoFiles??[],...p.SFiles??[]]){if(path.isAbsolute(f)){generated.push({path:f,sha256:hash(fs.readFileSync(f))});continue;}paths.add(path.relative(repo,path.join(p.Dir,f)));}if(p.ImportPath==='go.temporal.io/server/tools/gomad3/deterministicio')for(const f of [...p.TestGoFiles??[],...p.XTestGoFiles??[],...p.TestEmbedFiles??[],...p.XTestEmbedFiles??[]])paths.add(path.relative(repo,path.join(p.Dir,f)));}
const descriptor='tools/gomad3/toolchain/version/descriptor.go';
const oldDescriptor=git('show','ca6fd855868fac364b69cb31394c87ad2912e623:'+descriptor),currentDescriptor=fs.readFileSync(descriptor,'utf8');
if(hash(oldDescriptor)!==oldManifest.get(descriptor))throw Error('old descriptor provenance mismatch');
const withoutGuide=s=>{const start=s.indexOf('func renderUpgradeGuide('),end=s.indexOf('\nfunc ',start+1);if(start<0||end<0)throw Error('guide function boundary absent');return s.slice(0,start)+s.slice(end);};
if(withoutGuide(oldDescriptor)!==withoutGuide(currentDescriptor))throw Error('runtime descriptor source differs');
const bound=[...paths].sort().map(p=>({path:p,sha256:hash(fs.readFileSync(p)),predecessor_sha256:oldManifest.get(p),exact:hash(fs.readFileSync(p))===oldManifest.get(p)}));
if(bound.some(p=>!p.exact&&p.path!==descriptor))throw Error('runtime/test dependency drift '+JSON.stringify(bound.filter(p=>!p.exact)));
for(const p of bound.filter(p=>p.path!==descriptor))if(/\bgomadversion\.(Generate|GeneratedFiles)\b/.test(fs.readFileSync(p.path,'utf8')))throw Error('guide generator reachable from reused closure '+p.path);
const names=['accepted-deterministicio-workspace-complement','accepted-cache-tmpfs'];
const packets=names.map(n=>{const file=path.join(predecessor,n+'-receipt.json'),receipt=JSON.parse(fs.readFileSync(file));if(receipt.exit_code!==0||receipt.source_tree_sha256!=='1ea646fc66ac63aa0d2dde38e434f8b5529c28ac9040b55b7a04c56c4ebc0f3f')throw Error('predecessor receipt invalid');if(hash(fs.readFileSync(path.join(predecessor,receipt.log)))!==receipt.log_sha256)throw Error('predecessor raw log drift');return {receipt:path.relative(repo,file),receipt_sha256:hash(fs.readFileSync(file)),command:receipt.command,source_tree_sha256:receipt.source_tree_sha256,counts:receipt.counts,log_sha256:receipt.log_sha256};});
const result={dependency_listing:'reuse-deterministicio-dependencies-receipt.json',dependency_listing_sha256:hash(source),predecessor_manifest:path.relative(repo,manifestPath),predecessor_manifest_sha256:hash(fs.readFileSync(manifestPath)),dependency_paths:bound,descriptor_exception:{path:descriptor,old_sha256:hash(oldDescriptor),current_sha256:hash(currentDescriptor),all_bytes_outside_unexported_renderUpgradeGuide_exact:true,generated_identity_file_exact:true,generator_references_absent_from_reused_sources:true,fresh_guide_proof:['final-version-receipt.json','walk-adapter-apply-receipt.json','walk-validate-receipt.json']},packets,coverage:'Unchanged original selected source assertions and admitted predecessor filesystem-separated union only; no new execution, cleanup admission, native wrapper, supplied SDK checkout or full-native-host credit.',mapping:path.relative(repo,path.join(predecessor,'assertion-mapping.md')),selection:path.relative(repo,path.join(predecessor,'deterministicio-selection.json'))};
result.generated_test_main=generated;
fs.writeFileSync(path.join(out,'predecessor-source-reuse.json'),JSON.stringify(result,null,2)+'\n');
console.log(JSON.stringify({dependency_paths:bound.length,exact_paths:bound.filter(p=>p.exact).length,packets:packets.length}));
