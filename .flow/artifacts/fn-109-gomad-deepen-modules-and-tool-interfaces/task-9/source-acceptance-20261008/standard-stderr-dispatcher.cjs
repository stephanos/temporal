#!/usr/bin/node
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const root=process.env.TASK9_STANDARD_STDERR_CONTROL,size=Number(process.env.TASK9_STANDARD_STDERR_SIZE),argv=process.argv.slice(2),hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex');
if(!root||![65535,65536,65537,4*1024*1024].includes(size)||JSON.stringify(argv)!==JSON.stringify(['list','std'])){process.stderr.write('unexpected standard stderr command');process.exit(97)}
const bytes=fs.readFileSync(path.join(root,'payload-'+size));
if(bytes.length!==size)throw Error('declared payload size changed');
const record={argv,cwd:process.cwd(),pid:process.pid,size,environment_sha256:hash(JSON.stringify(Object.entries(process.env).sort()))};
fs.appendFileSync(path.join(root,'commands.jsonl'),JSON.stringify(record)+'\n');
const wait=new Int32Array(new SharedArrayBuffer(4));let written=0;
while(written<bytes.length){let count;try{count=fs.writeSync(2,bytes,written,bytes.length-written)}catch(error){if(error.code!=='EAGAIN')throw error;Atomics.wait(wait,0,0,1);continue}if(count<=0)throw Error('output write made no progress');written+=count;}
fs.appendFileSync(path.join(root,'outputs.jsonl'),JSON.stringify({...record,stderr_bytes:written,stderr_sha256:hash(bytes),declared_exit:7})+'\n');
process.exit(7);
