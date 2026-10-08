import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.join(repo, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/source-acceptance-20261008');
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const env = {...process.env, PATH:path.dirname(go)+':'+process.env.PATH, GOENV:'off', GOWORK:'off', GOTOOLCHAIN:'local', GOFLAGS:'', GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r', TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOMODCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc', GOLANGCI_LINT_CACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/.lint-cache'};
for (const key of ['BASH_ENV','GOROOT','GOBIN','GOMADSEED','GOMAD3_CHILD_SEED']) delete env[key];
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const git = (...args) => {const r=spawnSync('git',args,{cwd:repo,encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);return r.stdout;};
const bindings = () => [...new Set([...git('ls-files','tools/gomad3','go.mod','go.sum','tools/gomad3integration/go.mod','tools/gomad3integration/go.sum','.github/.golangci.yml','Makefile','cmd/tools/lintcode').split('\n'),...git('ls-files','--others','--exclude-standard','tools/gomad3').split('\n')].filter(Boolean))].sort().map(p=>({path:p,sha256:fs.existsSync(path.join(repo,p))?hash(fs.readFileSync(path.join(repo,p))):null}));
const toolBindings = () => [go,path.join(path.dirname(go),'gofmt'),'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','/tmp/fn109-lint-tools.ZdNe1t50/errortype'].map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
const controlBindings = () => {
 const paths = new Set();
 for (const name of ['source-controls.json','caller-controls.json','caller-fixed-inputs.json']) {
  const file=path.join(out,name);if(!fs.existsSync(file))continue;paths.add(file);
  const control=JSON.parse(fs.readFileSync(file));
  if(control.overlay){paths.add(control.overlay);for(const [source,overlay] of Object.entries(JSON.parse(fs.readFileSync(control.overlay)).Replace)){paths.add(source);paths.add(overlay);}}
  for(const input of control.inputs??[])if(input.path)paths.add(input.path);
  if(control.original){const walk=dir=>fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(dir,e.name)):[path.join(dir,e.name)]);for(const file of walk(control.original))paths.add(file);}
 }
 const owner=path.join(out,'owner-controls.json');
 if(fs.existsSync(owner)){paths.add(owner);for(const input of JSON.parse(fs.readFileSync(owner))){for(const key of ['source','overlay','manifest'])if(input[key])paths.add(input[key]);}}
 return [...paths].sort().map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
};
const [name,command] = process.argv.slice(2);
if (!name || !command || !/^[a-z0-9-]+$/.test(name)) throw Error('run.mjs NAME COMMAND');
if(fs.existsSync(path.join(out,name+'-receipt.json'))) throw Error('receipt already exists');
const initial=bindings(), tools=toolBindings(), sourceHash=hash(JSON.stringify(initial));
const controls=controlBindings(),controlHash=hash(JSON.stringify(controls)),controlManifest='controls-'+controlHash+'.json';
if(!fs.existsSync(path.join(out,controlManifest)))fs.writeFileSync(path.join(out,controlManifest),JSON.stringify(controls,null,2)+'\n');
const manifest='source-'+sourceHash+'.json';
if(!fs.existsSync(path.join(out,manifest)))fs.writeFileSync(path.join(out,manifest),JSON.stringify(initial,null,2)+'\n');
const log=path.join(out,name+'.log'),fd=fs.openSync(log,'wx'),started=new Date().toISOString(),start=performance.now();
const r=spawnSync('timeout',['600','bash','-c','test "$(pwd -P)" = '+repo+' || exit 99\n'+command],{cwd:repo,env,stdio:['ignore',fd,fd]});
fs.closeSync(fd);
const events=fs.readFileSync(log,'utf8').split('\n').filter(s=>s.startsWith('{')).flatMap(s=>{try{return [JSON.parse(s)];}catch{return [];}});
const receipt={name,command,started,ended:new Date().toISOString(),elapsed_seconds:(performance.now()-start)/1000,exit_code:r.status,signal:r.signal,error:r.error?.message??null,log:path.basename(log),log_sha256:hash(fs.readFileSync(log)),base_commit:'b4602685b3184387cf2d713178095247f0c11d8f',head:git('rev-parse','HEAD').trim(),source_manifest:manifest,source_tree_sha256:sourceHash,source_unchanged:sourceHash===hash(JSON.stringify(bindings())),tools,tools_unchanged:JSON.stringify(tools)===JSON.stringify(toolBindings()),environment:Object.fromEntries(Object.entries(env).filter(([k])=>['PATH','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','GOCACHE','GOMODCACHE','GOLANGCI_LINT_CACHE','TMPDIR','GOTMPDIR','GOPROXY','GOPRIVATE','GONOSUMDB','GOSUMDB'].includes(k))),counts:Object.fromEntries(['pass','fail','skip'].map(action=>[action,events.filter(e=>e.Action===action&&e.Test).length])),observations:events.filter(e=>['pass','fail','skip'].includes(e.Action)&&e.Test),harness_sha256:hash(fs.readFileSync(path.join(out,'run.mjs')))};
receipt.control_manifest=controlManifest;receipt.control_tree_sha256=controlHash;receipt.controls_unchanged=controlHash===hash(JSON.stringify(controlBindings()));
fs.writeFileSync(path.join(out,name+'-receipt.json'),JSON.stringify(receipt,null,2)+'\n');
console.log(JSON.stringify({name,exit_code:r.status,source_tree_sha256:sourceHash,source_unchanged:receipt.source_unchanged,counts:receipt.counts}));
if(!receipt.source_unchanged||!receipt.tools_unchanged||!receipt.controls_unchanged)throw Error('source, tools or controls changed during gate');
if(r.status!==0)process.exitCode=1;
