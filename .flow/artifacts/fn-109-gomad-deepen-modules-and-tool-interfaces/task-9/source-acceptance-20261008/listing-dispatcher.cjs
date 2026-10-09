#!/usr/bin/node
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const root=process.env.TASK9_LISTING_CONTROL,argv=process.argv.slice(2),mode=process.env.TASK9_LISTING_MODE,start=performance.now();
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const standard=JSON.stringify(argv)===JSON.stringify(['list','std']),review=JSON.stringify(argv)===JSON.stringify(['list','-deps','-json','-mod=readonly','.']);
if(!root||(!standard&&!review)){process.stderr.write('unexpected listing command '+JSON.stringify(argv));process.exit(97)}
const record={argv,cwd:process.cwd(),environment:Object.fromEntries(Object.entries(process.env).filter(([k])=>k.startsWith('GO')||k.startsWith('TASK9')||['CGO_ENABLED','PATH','TZ','TEST_TELEMETRY_DIR','TMPDIR'].includes(k))),environment_sha256:hash(JSON.stringify(Object.entries(process.env).sort())),pid:process.pid,mode,standard};
fs.appendFileSync(path.join(root,'commands.jsonl'),JSON.stringify(record)+'\n');
if(mode==='startup'){process.stderr.write('declared startup rejection');process.exit(97)}
let stdout=standard?'fmt\n':JSON.stringify({ImportPath:'example.com/task9',Name:'main',Dir:path.join(root,'module'),GoFiles:['main.go'],Imports:[],Module:{Path:'example.com/task9',Main:true}}),stderr='listing diagnostic',code=0;
if(mode==='nonzero'){code=7}else if(mode==='invalid'){stderr='no required module provides package';code=7}else if(mode==='malformed'){stdout='{broken'}
if(mode==='stdout-overflow')stdout='x'.repeat((standard?4:64)*1024*1024+1);
if(mode==='stderr-overflow')stderr='x'.repeat((standard?4:64)*1024*1024+1);
fs.writeSync(1,stdout);fs.writeSync(2,stderr);
fs.appendFileSync(path.join(root,'outputs.jsonl'),JSON.stringify({pid:process.pid,mode,standard,stdout_bytes:Buffer.byteLength(stdout),stdout_sha256:hash(stdout),stderr_bytes:Buffer.byteLength(stderr),stderr_sha256:hash(stderr),declared_exit:code,elapsed_seconds:(performance.now()-start)/1000})+'\n');
if(['cancel','deadline'].includes(mode)){fs.writeFileSync(path.join(root,'marker'),String(process.pid));setInterval(()=>{},1000)}else if(mode==='signal'){process.kill(process.pid,'SIGKILL')}else{process.exit(code)}
