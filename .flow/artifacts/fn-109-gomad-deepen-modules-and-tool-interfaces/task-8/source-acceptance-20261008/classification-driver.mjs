#!/usr/bin/node
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
let root=path.resolve(process.argv[1],'../..');
if(path.basename(path.dirname(root))==='builds')root=path.resolve(root,'../..');
const args=process.argv.slice(2);
const go='/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
if(args[0]==='list'||args[0]==='env'){
 const result=spawnSync(go,args,{stdio:'inherit',env:{...process.env,GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',GOOS:'linux',GOARCH:'amd64'}});
 process.exit(result.status??98);
}
if(args[0]!=='build')throw Error('SOURCE unexpected command or target launch '+JSON.stringify(args));
const output=args[4],base='/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX';
const want=['build','-trimpath','-buildvcs=false','-o',output,'-gcflags=all=-gomadcap','-ldflags=-linkmode=internal -gomadcap='+ '7'.repeat(64),'.'];
const linked=path.dirname(output),owned=path.dirname(linked);
if(JSON.stringify(args)!==JSON.stringify(want)||path.dirname(owned)!==base||!/^gomad3-(analysis|compatibility-review)-/.test(path.basename(owned))||!/^\.linked-review-/.test(path.basename(linked))||path.basename(output)!=='target'||fs.realpathSync(linked)!==linked||process.cwd()!==fs.readFileSync(path.join(root,'allowed-working'),'utf8'))throw Error('SOURCE unexpected build or output path '+JSON.stringify(args));
fs.writeFileSync(output,'SOURCE-malformed-linked-object\n',{flag:'wx',mode:0o600});
const bytes=fs.readFileSync(output);
fs.writeFileSync(path.join(root,'build-command.json'),JSON.stringify({commands:[{command:[process.argv[1],...args],output,output_bytes:bytes.toString(),output_sha256:crypto.createHash('sha256').update(bytes).digest('hex'),output_length:bytes.length}]}),{flag:'wx'});
