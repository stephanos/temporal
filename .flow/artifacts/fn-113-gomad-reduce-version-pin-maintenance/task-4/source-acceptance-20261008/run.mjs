import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

export const repo = process.cwd();
export const out = path.join(repo, '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/source-acceptance-20261008');
export const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const cache = '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc';
export const env = {...process.env, PATH:path.dirname(go)+':'+process.env.PATH, GOENV:'off', GOWORK:'off', GOTOOLCHAIN:'local', GOFLAGS:'', GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r', TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOMODCACHE:cache, GOLANGCI_LINT_CACHE:path.join(cache,'.lint-cache'), TEST_TELEMETRY_DIR:path.join(cache,'.go-telemetry')};
export const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
export const git = (...args) => {const r=spawnSync('git',args,{cwd:repo,encoding:'utf8',maxBuffer:16<<20}); if(r.status!==0) throw Error(r.error?.message??r.stderr); return r.stdout;};
export const bindings = () => [...new Set([...git('ls-files','tools/gomad3','go.mod','go.sum','tools/gomad3integration/go.mod','tools/gomad3integration/go.sum','.github/.golangci.yml','Makefile','cmd/tools/lintcode','MILESTONES.md','AGENTS.md','.plans/GOMAD_NEXT.md').split('\n'),...git('ls-files','--others','--exclude-standard','tools/gomad3').split('\n')].filter(Boolean))].sort().map(p=>({path:p,sha256:fs.existsSync(p)?hash(fs.readFileSync(p)):null}));
const toolBindings = () => [go,path.join(path.dirname(go),'gofmt'),'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0','/tmp/fn109-lint-tools.ZdNe1t50/errortype'].map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
export function run(name, command, expected='0', options={}) {
  const initial=bindings(), tools=toolBindings(), sourceHash=hash(JSON.stringify(initial));
  const manifest='source-'+sourceHash+'.json';
  if(!fs.existsSync(path.join(out,manifest))) fs.writeFileSync(path.join(out,manifest),JSON.stringify(initial,null,2)+'\n');
  const log=path.join(out,name+'.log');
  if(fs.existsSync(log)) throw Error('refusing to overwrite '+log);
  const fd=fs.openSync(log,'wx'), start=Date.now(), started=new Date().toISOString();
  const effectiveEnv={...env,...options.env};
  delete effectiveEnv.BASH_ENV;
  const expectedDirectory=options.cwd??repo;
  const assertedCommand='test "$(pwd -P)" = '+JSON.stringify(fs.realpathSync(expectedDirectory))+' || exit 125\n'+command;
  const r=spawnSync('timeout',['600','bash','-c',assertedCommand],{cwd:expectedDirectory,env:effectiveEnv,stdio:['ignore',fd,fd]});
  fs.closeSync(fd);
  const events=fs.readFileSync(log,'utf8').split('\n').filter(s=>s.startsWith('{')).flatMap(s=>{try{return [JSON.parse(s)];}catch{return [];}});
  const receipt={name,command,working_directory:options.cwd??repo,started,ended:new Date().toISOString(),elapsed_seconds:(Date.now()-start)/1000,exit_code:r.status,signal:r.signal,error:r.error?.message??null,log:path.basename(log),log_sha256:hash(fs.readFileSync(log)),base_commit:fs.readFileSync('.flow/tmp/base_commit','utf8').trim(),head:git('rev-parse','HEAD').trim(),source_tree_sha256:sourceHash,source_manifest:manifest,source_unchanged:sourceHash===hash(JSON.stringify(bindings())),tools,tools_unchanged:JSON.stringify(tools)===JSON.stringify(toolBindings()),environment:Object.fromEntries(Object.entries(effectiveEnv).filter(([k])=>['PATH','GOENV','GOWORK','GOTOOLCHAIN','GOFLAGS','GOCACHE','GOMODCACHE','GOLANGCI_LINT_CACHE','TEST_TELEMETRY_DIR','TMPDIR','GOTMPDIR','GOOS','GOARCH','CGO_ENABLED'].includes(k))),counts:Object.fromEntries(['pass','fail','skip'].map(action=>[action,events.filter(e=>e.Action===action&&e.Test).length])),skips:events.filter(e=>e.Action==='skip'),failures:events.filter(e=>e.Action==='fail'),harness_sha256:hash(fs.readFileSync(path.join(out,'run.mjs')))};
  receipt.shell_environment={BASH_ENV:'unset in child',removed_inherited_BASH_ENV:env.BASH_ENV??null,cwd_assertion:fs.realpathSync(expectedDirectory)};
  if(effectiveEnv.XDG_CACHE_HOME) receipt.environment.XDG_CACHE_HOME=effectiveEnv.XDG_CACHE_HOME;
  if(/make -C tools\/gomad3 (validate|generate)/.test(command)) receipt.effective_generator_cache=path.join(repo,'tools/gomad3/.toolchain/generator-cache');
  fs.writeFileSync(path.join(out,name+'-receipt.json'),JSON.stringify(receipt,null,2)+'\n');
  console.log(JSON.stringify({name,exit_code:r.status,elapsed_seconds:receipt.elapsed_seconds,source_tree_sha256:sourceHash,source_unchanged:receipt.source_unchanged,counts:receipt.counts}));
  if(!receipt.source_unchanged && !options.mutatesSource || !receipt.tools_unchanged) throw Error('source or tools changed during gate');
  if(expected!=='any' && r.status!==Number(expected)) throw Error('unexpected exit; read '+log);
  return receipt;
}
if(process.argv[1]===path.join(out,'run.mjs')) {
  const [name,command,expected='0'] = process.argv.slice(2);
  if(!name || !command) throw Error('run.mjs NAME COMMAND [EXPECTED|any]');
  run(name,command,expected,{mutatesSource:name==='generate'});
}
