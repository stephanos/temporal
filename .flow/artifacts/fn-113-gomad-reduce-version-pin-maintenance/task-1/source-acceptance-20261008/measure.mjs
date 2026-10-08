import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo=process.cwd(), root=path.join(repo,'tools/gomad3');
const output=path.join(repo,'.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/source-acceptance-20261008');
const digest=b=>crypto.createHash('sha256').update(b).digest('hex');
const read=p=>fs.readFileSync(p);
const json=p=>JSON.parse(read(p));
const walk=p=>fs.readdirSync(p,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(p,e.name)):[path.join(p,e.name)]).sort();
const lines=b=>b.length===0?0:b.toString().split('\n').length-(b.at(-1)===10?1:0);
const version=json(path.join(root,'toolchain/version/version.json'));
const boundary=json(path.join(root,'deterministicio/boundary/manifest.json'));
const packPaths=fs.readdirSync(path.join(root,'internal/compatibilitypack/packs')).filter(p=>p.endsWith('.json')).map(p=>path.join(root,'internal/compatibilitypack/packs',p)).sort();
const packs=packPaths.map(json);
const adapterPaths=fs.readdirSync(path.join(root,'deterministicio')).filter(p=>p.endsWith('_adapter.go')).map(p=>path.join(root,'deterministicio',p)).sort();
const overlay=walk(path.join(root,'toolchain/runtime/overlay'));
const patch=read(path.join(root,version.patch));
const clock=read(path.join(root,'toolchain/clock_inventory_test.go')).toString();
const clockLiteral=clock.split('var reviewedHostClockReferences = []clockReference{')[1]?.split('\n}\n')[0];
if(!clockLiteral) throw Error('clock inventory literal absent');
const clockRows=[...clockLiteral.matchAll(/^\s*\{"(darwin\/arm64|linux\/amd64)", "[^"]+", "[^"]+", (\d+), /gm)];
const counts={
  runtime_patch:{lines:lines(patch),bytes:patch.length,files:[...patch.toString().matchAll(/^--- a\//gm)].length},
  runtime_overlay:{files:overlay.length,lines:overlay.reduce((n,p)=>n+lines(read(p)),0)},
  dependency_adapters:{modules:version.adapters.length,sha256_anchor_literals:adapterPaths.reduce((n,p)=>n+[...read(p).toString().matchAll(/"sha256:[0-9a-f]{64}"/g)].length,0)},
  compatibility_packs:{packs:packs.length,rules:packs.reduce((n,p)=>n+p.rules.length,0),unique_module_version_pins:new Set(packs.flatMap(p=>[...p.activation,...p.rules.map(r=>r.module)]).map(m=>m.path+'@'+m.version)).size},
  interception_fingerprints:{intercepts:boundary.intercepts.length,declarations_including_platform_overrides:boundary.intercepts.reduce((n,e)=>n+1+Object.keys(e.platform_overrides??{}).length,0)},
  clock_inventory:{references:clockRows.length,by_platform:Object.fromEntries(['darwin/arm64','linux/amd64'].map(p=>[p,clockRows.filter(r=>r[1]===p).length])),identifier_occurrences:clockRows.reduce((n,r)=>n+Number(r[2]),0)},
};
const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/fn1131-inventory-');
const program=path.join(scratch,'inventory.go');
fs.writeFileSync(program,`package main
import("context";"fmt";"os";"path/filepath";"go.temporal.io/server/tools/gomad3/upgrade/pinimpact";"golang.org/x/mod/modfile")
type resolver struct{}
func(resolver)Resolve(_ context.Context,f pinimpact.ModuleFiles)(map[string]string,error){m,e:=modfile.Parse("go.mod",f.GoMod,nil);if e!=nil{return nil,e};r:=map[string]string{};for _,v:=range m.Require{r[v.Mod.Path]=v.Mod.Version};return r,nil}
func main(){r:=os.Args[1];m,e:=os.ReadFile(filepath.Join(r,"go.mod"));if e!=nil{panic(e)};s,e:=os.ReadFile(filepath.Join(r,"go.sum"));if e!=nil{panic(e)};f:=pinimpact.ModuleFiles{GoMod:m,GoSum:s,Directory:r};v,e:=pinimpact.Evaluate(context.Background(),pinimpact.Spec{Root:filepath.Join(r,"tools/gomad3"),Baseline:f,Candidate:f,Resolver:resolver{},IncludeAll:true});if e!=nil{panic(e)};b,e:=pinimpact.Encode(v);if e!=nil{panic(e)};fmt.Print(string(b))}
`);
const env={...process.env,PATH:'/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:'+process.env.PATH,GOENV:'off',GOWORK:'off',GOTOOLCHAIN:'local',GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX'};
const command=['600','go','-C','tools/gomad3','run',program,repo], started=Date.now();
const run=spawnSync('timeout',command,{cwd:repo,env,encoding:'utf8',maxBuffer:2*1024*1024});
if(run.status!==0) throw Error(run.stderr);
fs.writeFileSync(path.join(scratch,'report.json'),run.stdout);
const report=JSON.parse(run.stdout);
for(const [cls,want] of [['adapter',counts.dependency_adapters.modules],['pack-rule',counts.compatibility_packs.rules],['interception-fingerprint',counts.interception_fingerprints.declarations_including_platform_overrides],['clock-reference',counts.clock_inventory.references]]) {
  if(report.pins.filter(p=>p.class===cls).length!==want) throw Error('report/count mismatch '+cls);
}
const bindings=[path.join(root,version.patch),path.join(root,'toolchain/version/version.json'),path.join(root,'deterministicio/boundary/manifest.json'),path.join(root,'toolchain/clock_inventory_test.go'),...packPaths,...adapterPaths];
const original='.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/baseline.json';
const measured={schema:'gomad3.pin-maintenance-baseline/v1',head:fs.readFileSync('.flow/tmp/base_commit','utf8').trim(),recorded_utc:new Date().toISOString(),measurement_command:'node '+path.relative(repo,import.meta.filename),counts,report_classes:report.classes,inventory_report_sha256:digest(run.stdout),inventory_command:['timeout',...command],inventory_command_exit:run.status,inventory_command_elapsed_seconds:(Date.now()-started)/1000,inventory_program_sha256:digest(read(program)),resolution:'Descriptor-only inventory uses direct go.mod requirements and immutable go.sum; it does not claim selected module-graph resolution or native build preparation.',historical_first_baseline:{path:original,sha256:digest(read(original)),head:json(original).head},overlay_tree_sha256:digest(overlay.map(p=>path.relative(root,p)+'\0'+digest(read(p))).join('\n')),source_sha256:Object.fromEntries(bindings.map(p=>[path.relative(repo,p),digest(read(p))]))};
fs.writeFileSync(path.join(output,'current-inventory.json'),JSON.stringify(measured,null,2)+'\n');
console.log(JSON.stringify(counts));
