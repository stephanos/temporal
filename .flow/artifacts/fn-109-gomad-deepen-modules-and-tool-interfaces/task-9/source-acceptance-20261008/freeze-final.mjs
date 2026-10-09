import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.dirname(new URL(import.meta.url).pathname),scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-final-controls-'),current=path.join(scratch,'current'),hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex');
const list=spawnSync('git',['ls-files','tools/gomad3','tools/gomad3sim','go.mod','go.sum'],{cwd:repo,encoding:'utf8'}),extra=spawnSync('git',['ls-files','--others','--exclude-standard','tools/gomad3'],{cwd:repo,encoding:'utf8'});
if(list.status!==0||extra.status!==0)throw Error('source enumeration failed');
const files=[];
for(const relative of [...new Set((list.stdout+extra.stdout).trim().split('\n'))].sort()){const source=path.join(repo,relative),file=path.join(current,relative);if(!fs.existsSync(source))continue;fs.mkdirSync(path.dirname(file),{recursive:true});fs.copyFileSync(source,file);files.push({path:file,source,sha256:hash(fs.readFileSync(file))});}
for(const [input,name]of [['listing-control_test.go.txt','task9_listing_control_test.go'],['standard-stderr-control_test.go.txt','task9_standard_stderr_control_test.go']]){const bytes=fs.readFileSync(path.join(out,input)),file=path.join(current,'tools/gomad3/target',name);fs.writeFileSync(file,bytes,{flag:'wx'});files.push({path:file,sha256:hash(bytes)});}
const probe=path.join(path.dirname(out),'adapter-command-gap-2026-10-05/base-process-probe/probe.go'),normalizer=path.join(path.dirname(out),'adapter-integration-20261005/verify-probes.cjs');
for(const name of ['adapter-original-root','adapter-current-root'])fs.mkdirSync(path.join(scratch,name),{mode:0o700});
const history=JSON.parse(fs.readFileSync(path.join(out,'historical-inputs.json'))),metadata={scratch,current,original:history.original,files,probe:{path:probe,sha256:hash(fs.readFileSync(probe))},normalizer:{path:normalizer,sha256:hash(fs.readFileSync(normalizer))},adapter_cwd:path.join(repo,'tools/gomad3'),scope:'Whole coherent frozen corrected-current source graph plus identical additive listing controls; unchanged original graph separately bound. Ordinary stock probe compiler outputs become explicit immutable run inputs only after compilation.'};
fs.writeFileSync(path.join(out,'final-control-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify({scratch,current,files:files.length}));
