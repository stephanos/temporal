import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(new URL(import.meta.url).pathname);
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const env = {...process.env, PATH:path.dirname(go)+':'+process.env.PATH, GOENV:'off', GOWORK:'off', GOTOOLCHAIN:'local', GOFLAGS:'', GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r', TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOMODCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc', GOLANGCI_LINT_CACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/.lint-cache', GOPROXY:'off'};
for (const key of ['BASH_ENV','GOROOT','GOBIN','GOMADSEED','GOMAD3_CHILD_SEED']) delete env[key];
env.TEST_TELEMETRY_DIR='/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-telemetry.98tay7';
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const git = (...args) => {const r=spawnSync('git',args,{cwd:repo,encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);return r.stdout;};
const bindings = () => [...new Set([...git('ls-files','tools/gomad3','go.mod','go.sum','tools/gomad3integration/go.mod','tools/gomad3integration/go.sum','.github/.golangci.yml','Makefile','cmd/tools/lintcode').split('\n'),...git('ls-files','--others','--exclude-standard','tools/gomad3').split('\n')].filter(Boolean))].sort().map(p=>({path:p,sha256:fs.existsSync(path.join(repo,p))?hash(fs.readFileSync(path.join(repo,p))):null}));
const tools = () => [go,path.join(path.dirname(go),'gofmt'),...['compile','link','asm','vet'].map(name=>path.join(path.dirname(path.dirname(go)),'pkg/tool/linux_arm64',name)),'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','/tmp/fn109-lint-tools.ZdNe1t50/errortype',process.execPath,'/usr/bin/bash','/usr/bin/timeout','/usr/bin/git','/usr/bin/tar'].map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
const controls = () => {
 const paths=new Set([path.join(out,'run.mjs')]);
 const file=path.join(out,'controls.json');
 if(fs.existsSync(file)) {
  paths.add(file);const config=JSON.parse(fs.readFileSync(file));for(const p of config.paths)paths.add(p);
  for(const manifest of config.historical_manifests??[]) {
   paths.add(manifest);const data=JSON.parse(fs.readFileSync(manifest));
   for(const files of Object.values(data.sources))for(const entry of files)paths.add(entry.path);
  }
  for(const manifest of [...(config.file_manifests??[]),...(config.versioned_file_manifests??[]),...(config.adapter_base_manifest?[config.adapter_base_manifest]:[])]) {
   paths.add(manifest);for(const entry of JSON.parse(fs.readFileSync(manifest)).files)paths.add(entry.path);
  }
 }
 return [...paths].sort().map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
};
const snapshot = (kind, entries) => {
 const bytes=JSON.stringify(entries,null,2)+'\n',digest=hash(bytes),name=kind+'-'+digest+'.json',file=path.join(out,name);
 if(!fs.existsSync(file))fs.writeFileSync(file,bytes,{flag:'wx'});
 for(const entry of entries.filter(x=>x.path.endsWith('.json')&&x.path.startsWith(out))) {
  const bytes=fs.readFileSync(entry.path),file=path.join(out,'metadata-'+entry.sha256+'.json');
  if(!fs.existsSync(file))fs.writeFileSync(file,bytes,{flag:'wx'});
 }
 return {manifest:name,sha256:digest};
};
const [name, command]=process.argv.slice(2);
if(!name||!command||!/^[a-z0-9-]+$/.test(name))throw Error('run.mjs NAME COMMAND');
if(fs.existsSync(path.join(out,name+'-receipt.json')))throw Error('receipt already exists');
const initial=bindings(),toolInputs=tools(),controlInputs=controls();
const source=snapshot('source',initial),tool=snapshot('tools',toolInputs),control=snapshot('controls',controlInputs);
const stdout=path.join(out,name+'.stdout'),stderr=path.join(out,name+'.stderr');
const fds=[fs.openSync(stdout,'wx'),fs.openSync(stderr,'wx')];
const argv=['timeout','600','bash','-c','test "$(pwd -P)" = '+repo+' || exit 99\n'+command];
const started=new Date().toISOString(),start=performance.now();
const result=spawnSync(argv[0],argv.slice(1),{cwd:repo,env,stdio:['ignore',...fds]});
fds.forEach(fd=>fs.closeSync(fd));
const events=fs.readFileSync(stdout,'utf8').split('\n').filter(s=>s.startsWith('{')).flatMap(s=>{try{return [JSON.parse(s)];}catch{return [];}});
const receipt={name,argv,command,cwd:repo,started,ended:new Date().toISOString(),elapsed_seconds:(performance.now()-start)/1000,exit_code:result.status,signal:result.signal,error:result.error?.message??null,terminal_handle:'foreground spawnSync completed',stdout:{path:path.basename(stdout),sha256:hash(fs.readFileSync(stdout))},stderr:{path:path.basename(stderr),sha256:hash(fs.readFileSync(stderr))},base_commit:'29c80199cd',original_baseline:'d635e23f00d926a43b942f25a9d05bd0ccb72025',head:git('rev-parse','HEAD').trim(),source,tools:tool,controls:control,source_unchanged:JSON.stringify(initial)===JSON.stringify(bindings()),tools_unchanged:JSON.stringify(toolInputs)===JSON.stringify(tools()),controls_unchanged:JSON.stringify(controlInputs)===JSON.stringify(controls()),environment:Object.fromEntries(Object.entries(env).filter(([k])=>['PATH','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','GOCACHE','GOMODCACHE','GOLANGCI_LINT_CACHE','TMPDIR','GOTMPDIR','GOPROXY','GOPRIVATE','GONOSUMDB','GOSUMDB','CGO_ENABLED','GOOS','GOARCH'].includes(k))),counts:Object.fromEntries(['pass','fail','skip'].map(action=>[action,events.filter(e=>e.Action===action&&e.Test).length])),observations:events.filter(e=>['pass','fail','skip'].includes(e.Action)&&e.Test)};
receipt.environment.TEST_TELEMETRY_DIR=env.TEST_TELEMETRY_DIR;
receipt.environment_sha256=hash(JSON.stringify(Object.entries(env).sort(([a],[b])=>a.localeCompare(b))));
fs.writeFileSync(path.join(out,name+'-receipt.json'),JSON.stringify(receipt,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({name,exit_code:result.status,source:source.sha256,source_unchanged:receipt.source_unchanged,counts:receipt.counts}));
if(!receipt.source_unchanged||!receipt.tools_unchanged||!receipt.controls_unchanged)throw Error('source/tool/control changed during gate');
if(result.status!==0)process.exitCode=1;
