import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),owner=path.dirname(out);
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const scratch=fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-public-preservation-');
const module=path.join(scratch,'module'),root=path.join(scratch,'tools/gomad3/.toolchain'),key='9'.repeat(64);
fs.mkdirSync(module,{recursive:true});
fs.writeFileSync(path.join(module,'go.mod'),'module example.com/task9\n\ngo 1.26.4\n',{flag:'wx'});
fs.writeFileSync(path.join(module,'main.go'),'package main\nfunc main() {}\n',{flag:'wx'});
fs.copyFileSync(path.join(owner,'prepared-fixture.go.txt'),path.join(scratch,'fixture.go'),fs.constants.COPYFILE_EXCL);
const config={scratch,root,module,key,payload:path.join(scratch,'payload'),go:'/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go',commands:path.join(scratch,'commands.jsonl')};
fs.writeFileSync(path.join(scratch,'control.json'),JSON.stringify(config,null,2)+'\n',{flag:'wx'});
const dispatcher=fs.readFileSync(path.join(out,'public-control-dispatcher.cjs'));
for(const name of [path.join(root,'bin/go'),path.join(root,'builds',key,'bin/go')]) {
 fs.mkdirSync(path.dirname(name),{recursive:true});fs.writeFileSync(name,dispatcher,{flag:'wx',mode:0o700});
}
fs.writeFileSync(path.join(root,'build-key'),key+'\n',{flag:'wx',mode:0o600});
const history=JSON.parse(fs.readFileSync(path.join(out,'historical-inputs.json')));
const originalQueries=fs.readFileSync(path.join(history.original,'tools/gomad3/target/target.go'),'utf8');
const metadata={...config,fixture_sha256:hash(fs.readFileSync(path.join(scratch,'fixture.go'))),dispatcher_sha256:hash(dispatcher),sources:['go.mod','main.go'].map(file=>({path:path.join(module,file),sha256:hash(fs.readFileSync(path.join(module,file)))})),original_fixture_command_inventory:[['env','GOVERSION','GOOS','GOARCH','CGO_ENABLED'],['list','-deps','-json','-mod=readonly','.'],['build','-trimpath','-buildvcs=false','-o','OUTPUT','.']],original_only_queries_consumed:[],original_source_sha256:hash(originalQueries),historical_root_modules:Object.entries(history.sources).map(([graph,files])=>({graph,files:files.filter(x=>['go.mod','go.sum','tools/gomad3/go.mod','tools/gomad3/go.sum'].some(p=>x.path===path.join(graph==='original'?history.original:history.final,p)))})),scope:'Only build transport copies one once-compiled stock-Go INPUT; env/list execute actual pinned stock Go with requested cwd/environment. No native/patched qualification or binary launch.'};
fs.writeFileSync(path.join(out,'public-control-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({scratch,root,module,payload:config.payload}));
