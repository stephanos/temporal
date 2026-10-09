import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-build-outcomes-');
const root=path.join(scratch,'installation'),module=path.join(scratch,'module'),key='9'.repeat(64),go='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
fs.mkdirSync(module,{recursive:true});
fs.writeFileSync(path.join(module,'go.mod'),'module example.com/task9\n\ngo 1.26.4\n',{flag:'wx'});
fs.writeFileSync(path.join(module,'main.go'),'package main\nfunc main() {}\n',{flag:'wx'});
const dispatcher=fs.readFileSync(path.join(out,'build-control-dispatcher.cjs'));
for(const file of [path.join(root,'bin/go'),path.join(root,'builds',key,'bin/go')]){fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,dispatcher,{flag:'wx',mode:0o700});}
fs.writeFileSync(path.join(root,'build-key'),key+'\n',{flag:'wx',mode:0o600});
const config={scratch,root,module,key,go,commands:path.join(scratch,'commands.jsonl'),literal_inputs:[['go.mod','module example.com/task9\n\ngo 1.26.4\n'],['main.go','package main\nfunc main() {}\n']],dispatcher_sha256:hash(dispatcher),modes:['stdout','stderr','stderr-stdout','malformed','signal','before-cancel','cancel','deadline']};
fs.writeFileSync(path.join(scratch,'control.json'),JSON.stringify(config,null,2)+'\n',{flag:'wx'});
fs.writeFileSync(path.join(out,'build-control-inputs.json'),JSON.stringify({...config,files:[path.join(scratch,'control.json'),path.join(root,'bin/go'),path.join(root,'builds',key,'bin/go'),path.join(root,'build-key'),path.join(module,'go.mod'),path.join(module,'main.go')].map(file=>({path:file,sha256:hash(fs.readFileSync(file))}))},null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({scratch,config:path.join(scratch,'control.json')}));
