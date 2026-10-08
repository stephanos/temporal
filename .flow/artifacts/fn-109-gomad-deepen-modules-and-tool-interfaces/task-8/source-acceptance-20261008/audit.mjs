import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.dirname(new URL(import.meta.url).pathname);
const prefix=process.argv[2]??'final-bound';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const git=(...args)=>{const r=spawnSync('git',args,{cwd:repo,encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);return r.stdout};
function stream(name,marker){
 const receipt=JSON.parse(fs.readFileSync(path.join(out,name+'-receipt.json'))),bytes=fs.readFileSync(path.join(out,receipt.log));
 if(receipt.exit_code!==0||!receipt.source_unchanged||!receipt.controls_unchanged||!receipt.tools_unchanged||hash(bytes)!==receipt.log_sha256)throw Error('bad final receipt '+name);
 const groups=new Map();for(const line of bytes.toString().split('\n')){let e;try{e=JSON.parse(line)}catch{continue}if(e.Action==='output'&&e.Test){const key=e.Package+'|'+e.Test;groups.set(key,(groups.get(key)??'')+e.Output)}}
 const rows=[];for(const [test,text]of groups)for(const m of text.matchAll(new RegExp(marker+' (.+)\\n','g')))rows.push({test,...JSON.parse(m[1])});
 return rows.sort((a,b)=>a.test.localeCompare(b.test));
}
const old=stream('classification-original-observed-boundary','SOURCE_CLASSIFICATION'),now=stream('classification-current-observed-boundary','SOURCE_CLASSIFICATION');
if(old.length!==3||JSON.stringify(old)!==JSON.stringify(now))throw Error('literal classification observations differ');
const boundaries={original:stream('classification-original-observed-boundary','SOURCE_COMMAND_BOUNDARY'),current:stream('classification-current-observed-boundary','SOURCE_COMMAND_BOUNDARY')};
fs.writeFileSync(path.join(out,prefix+'-classification-comparison.json'),JSON.stringify({rows:3,literal_equal:true,original:old,current:now,boundaries,path_ledger_scope:'Raw independent toolchain/output paths are retained here, not normalized in public diagnostics.',native:false},null,2)+'\n',{flag:'wx'});
const parseLint=name=>{
 const text=fs.readFileSync(path.join(out,name+'.log'),'utf8');
 return [...text.matchAll(/^(tools\/gomad3\/[^:\n]+):(\d+):(\d+): (.+) \(([^()]+)\)\n([^\n]*)\n([^\n]*)/gm)].map(m=>({path:m[1],line:Number(m[2]),column:Number(m[3]),message:m[4],linter:m[5],source:m[6],raw:m[0]}));
};
const before=parseLint('lint-scoped-unfiltered'),after=parseLint('final-lint-scoped-unfiltered');
if(before.length!==125||after.length!==119)throw Error('unexpected lint inventory');
const counts={};for(const f of after)counts[f.path]=(counts[f.path]??0)+1;
const identity=f=>JSON.stringify([f.path,f.message,f.linter,f.source]);
const resolved=before.filter(f=>f.path==='tools/gomad3/cmd/gomadtool/compatibility_pack.go'&&[24,41,78,84,329,335].includes(f.line));
const remaining=after.map(identity);
for(const f of before.filter(f=>!resolved.includes(f))){const index=remaining.indexOf(identity(f));if(index<0)throw Error('unexpected removed residual '+JSON.stringify(f));remaining.splice(index,1)}
if(remaining.length||resolved.length!==6||resolved.some(f=>f.path!=='tools/gomad3/cmd/gomadtool/compatibility_pack.go'))throw Error('unexpected introduced/resolved lint sites');
const owner=f=>f.path.endsWith('/compatibility_pack.go')?'fn109.23/fn109.21: remaining developer diagnostics; authoring/task46 semantics unchanged':f.path.includes('/cmd/gomad/internal/cli/')?'fn109.9/fn109.23/fn109.21: existing host-command/CLI diagnostics': 'fn109.10/fn109.23/fn109.21: existing developer-command diagnostics';
fs.writeFileSync(path.join(out,prefix+'-lint-attribution.json'),JSON.stringify({before:125,after:119,resolved,by_file:counts,introduced:[],residuals:after.map(f=>({...f,owner:owner(f)})),supersedes:'final-lint-attribution.json resolved duplicate generic statements by multiset only, mislocated two fixes at416/421. This exact-site inventory binds the six admitted baseline sites and checks the remaining raw finding multiset unchanged.',limitations:'Configured diff-filtered make lint-code-fast passes; this unfiltered gate is RED. Residual ownership is carried work, not a waiver or full clean lint claim.'},null,2)+'\n',{flag:'wx'});
const preimages=JSON.parse(fs.readFileSync(path.join(out,'../preimages.json')));
for(const [file,digest]of Object.entries(preimages))if(hash(fs.readFileSync(path.join(out,'../preimages',file)))!==digest)throw Error('preimage mutated '+file);
const patch=path.join(out,'../task-only.patch');if(hash(fs.readFileSync(patch))!=='b8b51e55be68807d0a9b4f81601cfd02ecd3668c408f58dc7227f07ca667ae91')throw Error('historical patch mutated');
const protectedFiles={'.turbo/plans/gomad3-glossary-update.md':'97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188','.turbo/technical-debt.md':'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'};
for(const [file,digest]of Object.entries(protectedFiles))if(hash(fs.readFileSync(path.join(repo,file)))!==digest)throw Error('protected file changed '+file);
const changed=git('diff','--name-only','HEAD').trim().split('\n').filter(Boolean);
if(changed.some(p=>p.endsWith('.go')&&!p.startsWith('tools/gomad3/')))throw Error('unbound root Go changes');
const receipts=fs.readdirSync(out).filter(n=>n.endsWith('-receipt.json')).sort().map(file=>{
 const r=JSON.parse(fs.readFileSync(path.join(out,file))),log=path.join(out,r.log);if(hash(fs.readFileSync(log))!==r.log_sha256)throw Error('log mutated '+file);
 return {file,sha256:hash(fs.readFileSync(path.join(out,file))),name:r.name,command:r.command,exit_code:r.exit_code,counts:r.counts,source_tree_sha256:r.source_tree_sha256,source_unchanged:r.source_unchanged,controls_unchanged:r.controls_unchanged,tools_unchanged:r.tools_unchanged,log:r.log,log_sha256:r.log_sha256};
});
const authored=fs.readdirSync(out).filter(n=>n.endsWith('.mjs')||n.endsWith('.go')).sort().map(file=>({file,sha256:hash(fs.readFileSync(path.join(out,file)))}));
const bindings={head:git('rev-parse','HEAD').trim(),root_go_dependencies:'Tracked root Go sources are bound by unchanged HEAD; no tracked root Go file outside tools/gomad3 differs from HEAD. Module files and nested sources have per-file receipt manifests.',preimages,task_only_patch_sha256:hash(fs.readFileSync(patch)),protected_files:protectedFiles,authored,receipts};
fs.writeFileSync(path.join(out,prefix+'-evidence-audit.json'),JSON.stringify(bindings,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({classification_rows:old.length,literal_equal:true,lint_before:before.length,lint_after:after.length,resolved:resolved.length,receipts:receipts.length,head:bindings.head}));
