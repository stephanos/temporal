import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {repo,out,env,hash,git} from './run.mjs';
const documents=['tools/gomad3/README.md','tools/gomad3/SPEC.md','tools/gomad3/CLI.md','tools/gomad3/ARCHITECTURE.md','tools/gomad3/deterministicio/boundary/upgrade-go1.27.1.md','.plans/GOMAD_NEXT.md','MILESTONES.md'];
const read=p=>fs.readFileSync(path.join(repo,p),'utf8');
const unfenced=s=>{let fence=null;return s.split('\n').filter(line=>{const m=line.match(/^\s{0,3}(`{3,}|~{3,})/);if(m){if(!fence)fence=m[1];else if(m[1][0]===fence[0]&&m[1].length>=fence.length)fence=null;return false;}return !fence;}).join('\n');};
const anchors=file=>{const text=unfenced(fs.readFileSync(file,'utf8')),seen=new Map(),result=new Set();for(const m of text.matchAll(/^#{1,6}\s+(.+?)\s*#*\s*$/gm)){const slug=m[1].replace(/\[([^\]]+)\]\([^)]*\)/g,'$1').toLowerCase().replace(/[^\p{L}\p{N}_\- ]/gu,'').replaceAll(' ','-');const index=seen.get(slug)??0;seen.set(slug,index+1);result.add(slug+(index?'-'+index:''));}for(const m of text.matchAll(/<a\s+(?:id|name)=["']([^"']+)/g))result.add(m[1]);return result;};
const errors=[],links=[];
for(const file of documents)for(const m of unfenced(read(file)).matchAll(/\[([^\]]+)\]\(([^)\n]+)\)/g)){
  const destination=m[2];if(/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(destination))continue;
  const [target,fragment]=destination.split('#'),absolute=target?path.resolve(path.dirname(path.join(repo,file)),decodeURIComponent(target)):path.join(repo,file);
  const exists=fs.existsSync(absolute),resolves=exists&&(!fragment||!absolute.endsWith('.md')||anchors(absolute).has(decodeURIComponent(fragment)));
  const entry={file,destination,exists,fragment_resolves:resolves};links.push(entry);if(!resolves)errors.push(entry);
}
const cases=file=>[...read(file).slice(0,read(file).indexOf('\nfunc ',read(file).indexOf('switch '))).matchAll(/case "([a-z][a-z-]*)":/g)].map(m=>m[1]);
const dispatch={gomad:cases('tools/gomad3/cmd/gomad/internal/cli/cli.go'),gomadtool:cases('tools/gomad3/cmd/gomadtool/main.go')};
const table=(body,heading)=>[...body.split(heading)[1].split('\n### ')[0].matchAll(/^\| `([^`]+)` \|/gm)].map(m=>m[1]);
const cli=read('tools/gomad3/CLI.md'),spec=read('tools/gomad3/SPEC.md');
const index={gomad:table(cli,'### `gomad`'),gomadtool:table(cli,'### `gomadtool`')};
const normative={};
for(const tool of ['gomad','gomadtool']) {
  const identifier=tool==='gomad'?'COMMAND.GOMAD':'COMMAND.GOMADTOOL';
  const block=spec.split('### ['+identifier+']')[1].split('\n## ')[0].split('\n### ')[0];
  normative[tool]=[...block.matchAll(/^\| `\[[^\]]+\]` \| `([^`]+)` \|/gm)].map(m=>m[1]);
  for(const surface of [index[tool],normative[tool]])if(JSON.stringify([...new Set(surface.filter(c=>!c.includes(' ')))].sort())!==JSON.stringify([...dispatch[tool]].sort()))errors.push({tool,error:'literal inventory differs',dispatch:dispatch[tool],surface});
}
const source=read('tools/gomad3/cmd/gomadtool/compatibility_pack.go');
const actions=[...source.slice(source.indexOf('switch arguments[0]'),source.indexOf('\nfunc runCompatibilityPackDiscover')).matchAll(/case "([^" ]+)":/g)].map(m=>m[1]);
const setup=JSON.parse(fs.readFileSync(path.join(out,'walk-setup.json')));
const binaries={gomadtool:setup.binary,gomad:path.join(setup.work,'gomad')};
const actual=[];const flags=new Set(['provenance']);
for(const [tool,commands] of Object.entries(dispatch))for(const command of commands){
  const variants=command==='compatibility-pack'?actions:[null];
  for(const action of variants){
    const argv=[command,...action?[action]:[],'-h'];
    const childEnv={...env};delete childEnv.BASH_ENV;
    const r=spawnSync(binaries[tool],argv,{cwd:repo,env:childEnv,encoding:'utf8',timeout:15000,maxBuffer:2<<20});
    const output=(r.stdout??'')+(r.stderr??'');const name=tool+'-'+command+(action?'-'+action:'')+'.help.txt';
    fs.writeFileSync(path.join(out,name),output);
    const registered=[...output.matchAll(/^\s+-([a-z][a-z0-9-]*)(?:\s|$)/gm)].map(m=>m[1]);for(const f of registered)flags.add(f);
    actual.push({tool,argv,status:r.status,signal:r.signal,error:r.error?.message??null,flags:registered,log:name,sha256:hash(output),binary_sha256:hash(fs.readFileSync(binaries[tool]))});
    if(r.error || !registered.length&&command!=='checked-run')errors.push({tool,command,action,error:'help unavailable',status:r.status});
  }
}
const documented=[...new Set([...cli.matchAll(/--([a-z][a-z0-9-]*)/g)].map(m=>m[1]))].sort();
const missingFlags=documented.filter(f=>!flags.has(f));if(missingFlags.length)errors.push({error:'documented flags absent from actual help',missingFlags});
const before=git('show','da1e726eab2d7211ec854df2d20fc2625c0c1695:tools/gomad3/SPEC.md');
const ids=s=>[...s.matchAll(/\[([A-Z][A-Z0-9]*(?:\.[A-Z0-9]+)*)\]/g)].map(m=>m[1]);
const addition='COMMAND.GOMADTOOL.QUALIFICATION.MANIFEST.GENERATE';
const preserved=JSON.stringify(ids(spec).filter(x=>x!==addition))===JSON.stringify(ids(before));
if(!preserved)errors.push({error:'preexisting semantic identifiers changed'});
const report={documents:documents.map(p=>({path:p,sha256:hash(read(p))})),links,dispatch,index,normative,compatibility_pack_actions:actions,actual_help:actual,documented_flags:documented,missing_flags:missingFlags,preexisting_semantic_identifiers_preserved:preserved,intentional_added_identifier:addition,errors};
fs.writeFileSync(path.join(out,'documentation-audit.json'),JSON.stringify(report,null,2)+'\n');
console.log(JSON.stringify({documents:documents.length,links:links.length,commands:dispatch.gomad.length+dispatch.gomadtool.length,help_invocations:actual.length,errors}));
if(errors.length)process.exit(1);
