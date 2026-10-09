#!/usr/bin/node
const fs=require('node:fs'),path=require('node:path'),cp=require('node:child_process'),crypto=require('node:crypto');
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const config=JSON.parse(fs.readFileSync(process.env.TASK9_FAILURE_CONTROL)),argv=process.argv.slice(2),mode=process.env.TASK9_FAILURE_MODE;
const start=performance.now(),base={argv,cwd:process.cwd(),environment:Object.fromEntries(Object.entries(process.env).filter(([k])=>k.startsWith('GO')||k.startsWith('TASK9')||['CGO_ENABLED','PATH','TMPDIR','TZ'].includes(k))),pid:process.pid,started:new Date().toISOString()};
if(!config.modes.includes(mode))throw Error('undeclared case');
if(JSON.stringify(argv)===JSON.stringify(['env','GOVERSION','GOOS','GOARCH','CGO_ENABLED'])||JSON.stringify(argv)===JSON.stringify(['list','-deps','-json','-mod=readonly','.'])) {
 const result=cp.spawnSync(config.go,argv,{cwd:process.cwd(),env:process.env,maxBuffer:64*1024*1024}),index=fs.existsSync(config.commands)?fs.readFileSync(config.commands,'utf8').trim().split('\n').length:0;
 const stdout=result.stdout??Buffer.alloc(0),stderr=result.stderr??Buffer.alloc(0),stdoutPath=path.join(config.scratch,'command-'+index+'.stdout'),stderrPath=path.join(config.scratch,'command-'+index+'.stderr');
 fs.writeFileSync(stdoutPath,stdout,{flag:'wx'});fs.writeFileSync(stderrPath,stderr,{flag:'wx'});
 fs.appendFileSync(config.commands,JSON.stringify({...base,kind:'actual-stock-Go',exit_code:result.status,signal:result.signal,elapsed_seconds:(performance.now()-start)/1000,stdout:{path:stdoutPath,sha256:hash(stdout)},stderr:{path:stderrPath,sha256:hash(stderr)}})+'\n');
 process.stdout.write(stdout);process.stderr.write(stderr);process.exit(result.status??98);
}
if(argv.length!==6||JSON.stringify(argv.slice(0,4))!==JSON.stringify(['build','-trimpath','-buildvcs=false','-o'])||argv[5]!=='.')throw Error('undeclared command '+JSON.stringify(argv));
const writes=mode==='stdout'?[[1,'stdout diagnostic']]:mode==='stderr'?[[2,'stderr diagnostic']]:mode==='malformed'?[[2,'{not a compiler JSON diagnostic}']]:[[2,'stderr first\n'],[1,'stdout second\n']];
fs.appendFileSync(config.commands,JSON.stringify({...base,kind:'controlled-build-error-NOT-compilation',mode,ordered_writes:writes,intended_outcome:mode==='signal'?'SIGKILL':['cancel','deadline'].includes(mode)?'wait for caller lifetime':'exit7'})+'\n');
for(const [fd,bytes]of writes)fs.writeSync(fd,bytes);
fs.writeFileSync(process.env.TASK9_BUILD_MARKER,String(process.pid)+'\n',{flag:'wx'});
if(mode==='signal')process.kill(process.pid,'SIGKILL');
else if(['cancel','deadline'].includes(mode))setInterval(()=>{},1000);
else process.exit(7);
