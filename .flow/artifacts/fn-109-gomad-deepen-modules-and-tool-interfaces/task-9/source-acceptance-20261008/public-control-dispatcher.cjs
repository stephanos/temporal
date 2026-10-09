#!/usr/bin/node
const fs=require('node:fs'),path=require('node:path'),cp=require('node:child_process'),crypto=require('node:crypto');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
let cursor=path.dirname(fs.realpathSync(__filename));
while(!fs.existsSync(path.join(cursor,'control.json'))) {const next=path.dirname(cursor);if(next===cursor)throw Error('missing immutable control');cursor=next;}
const config=JSON.parse(fs.readFileSync(path.join(cursor,'control.json'))),argv=process.argv.slice(2),start=performance.now();
const env=Object.fromEntries(Object.entries(process.env).filter(([k])=>k.startsWith('GO')||['CGO_ENABLED','PATH','TMPDIR','TZ'].includes(k)));
let result,kind;
if(JSON.stringify(argv)===JSON.stringify(['env','GOVERSION','GOOS','GOARCH','CGO_ENABLED'])||JSON.stringify(argv)===JSON.stringify(['list','-deps','-json','-mod=readonly','.'])) {
 kind='actual-stock-Go';result=cp.spawnSync(config.go,argv,{cwd:process.cwd(),env:process.env,maxBuffer:64*1024*1024});
} else if(argv.length===6&&JSON.stringify(argv.slice(0,4))===JSON.stringify(['build','-trimpath','-buildvcs=false','-o'])&&argv[5]==='.') {
 kind='copy-immutable-binary-INPUT';fs.copyFileSync(config.payload,argv[4]);result={status:0,signal:null,stdout:Buffer.alloc(0),stderr:Buffer.alloc(0)};
} else {
 kind='rejected-unexpected-command';result={status:97,signal:null,stdout:Buffer.alloc(0),stderr:Buffer.from('unexpected task9 control command: '+JSON.stringify(argv)+'\n')};
}
const stdout=result.stdout??Buffer.alloc(0),stderr=result.stderr??Buffer.alloc(0),index=fs.existsSync(config.commands)?fs.readFileSync(config.commands,'utf8').trim().split('\n').length:0;
const stdoutPath=path.join(cursor,'command-'+index+'.stdout'),stderrPath=path.join(cursor,'command-'+index+'.stderr');
fs.writeFileSync(stdoutPath,stdout,{flag:'wx'});fs.writeFileSync(stderrPath,stderr,{flag:'wx'});
fs.appendFileSync(config.commands,JSON.stringify({index,executable:fs.realpathSync(__filename),argv,cwd:process.cwd(),environment:env,environment_sha256:hash(JSON.stringify(Object.entries(process.env).sort())),kind,exit_code:result.status,signal:result.signal,error:result.error?.message??null,elapsed_seconds:(performance.now()-start)/1000,stdout:{path:stdoutPath,sha256:hash(stdout)},stderr:{path:stderrPath,sha256:hash(stderr)},payload_sha256:fs.existsSync(config.payload)?hash(fs.readFileSync(config.payload)):null})+'\n');
process.stdout.write(stdout);process.stderr.write(stderr);process.exit(result.status??98);
