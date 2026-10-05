const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const crypto = require('node:crypto');
const cp = require('node:child_process');
const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const cwd = path.join(repo, 'tools/gomad3');
const root = __dirname;
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
const probe = path.join(root, '../adapter-command-gap-2026-10-05/base-process-probe/probe.go');
const env = {...process.env, PATH: path.dirname(go)+':/usr/local/bin:/usr/bin:/bin', GOENV:'off', GOWORK:'off', GOTOOLCHAIN:'local', GOPROXY:'off', GOSUMDB:'off', GOFLAGS:'', GOMAXPROCS:'2'};
for (const key of ['GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED']) delete env[key];
const git = (...args) => cp.execFileSync('git', args, {cwd:repo, encoding:'utf8'});
const sha = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const sources = git('ls-files','-z','tools/gomad3').split('\0').filter(Boolean);
const snapshot = () => ({head:git('rev-parse','HEAD').trim(), status:git('status','--short'), files:Object.fromEntries(sources.map(file=>[file,sha(path.join(repo,file))])), inputs:{go:sha(go), probe:sha(probe), runner:sha(__filename), measurement:sha(path.join(root,'source-selection.go'))}});
const write = (file,data) => fs.writeFileSync(file,data,{flag:'wx'});
function run(label,args,timeout=180000) {
  const out=path.join(root,label); fs.mkdirSync(out);
  const before=snapshot(); write(path.join(out,'before.json'),JSON.stringify(before,null,2)+'\n');
  const started=new Date().toISOString(), start=Date.now();
  const result=cp.spawnSync(go,args,{cwd,env,timeout,killSignal:'SIGKILL',maxBuffer:64*1024*1024});
  write(path.join(out,'stdout.txt'),result.stdout||Buffer.alloc(0));
  write(path.join(out,'stderr.txt'),result.stderr||Buffer.alloc(0));
  const after=snapshot(); write(path.join(out,'after.json'),JSON.stringify(after,null,2)+'\n');
  const receipt={label,command:[go,...args],cwd,started,finished:new Date().toISOString(),elapsed_ms:Date.now()-start,status:result.status,signal:result.signal,error:result.error?.message,host:{platform:os.platform(),arch:os.arch(),release:os.release()},environment:Object.fromEntries(['PATH','GOENV','GOWORK','GOTOOLCHAIN','GOPROXY','GOSUMDB','GOFLAGS','GOMAXPROCS','GOMODCACHE','GOCACHE','GOROOT','GOEXPERIMENT','CGO_ENABLED'].map(k=>[k,env[k]??null])),unset:['GOMADSEED','GOMAD3_CHILD_SEED','GOMAD3_SEED'],inputs_unchanged:JSON.stringify(before.files)===JSON.stringify(after.files)&&JSON.stringify(before.inputs)===JSON.stringify(after.inputs)&&before.head===after.head,stdout_sha256:sha(path.join(out,'stdout.txt')),stderr_sha256:sha(path.join(out,'stderr.txt'))};
  write(path.join(out,'receipt.json'),JSON.stringify(receipt,null,2)+'\n');
  console.log(JSON.stringify({label,status:receipt.status,elapsed_ms:receipt.elapsed_ms,inputs_unchanged:receipt.inputs_unchanged}));
  if(!receipt.inputs_unchanged) throw Error('source/input changed during '+label);
  return result;
}
if (snapshot().head !== 'd2e0e035519f1385b9acf630a70152655b113f61') throw Error('unexpected HEAD');
for(let n=1;n<=2;n++) {
  const scratch=fs.mkdtempSync(path.join(os.tmpdir(),'fn109-task9-base-process-'));
  const r=run('process-probe-'+n,['run','-tags','test_dep',probe,scratch],90000);
  if(r.status!==0) throw Error('process probe failed');
}
const selection=fs.mkdtempSync(path.join(os.tmpdir(),'fn109-task9-source-selection-'));
const measured=run('source-selection',['run','-tags','test_dep',path.join(root,'source-selection.go'),go,selection],180000);
if(measured.status!==0) throw Error('source measurement failed');
run('focused-command-packages',['test','-json','-count=1','-tags','test_dep','./internal/hostexec','./target/internal/gocommand','./target/internal/capabilityreview'],180000);
run('bounded-target-controls',['test','-json','-count=1','-tags','test_dep','./target','-run','^(TestTargetFileCleanup.*|TestPreparedCacheDigest.*|TestCapabilityReviewGoldenCanonicalBytes|TestCompatibilityPackProjectionPreserves.*|TestReadToolchainIdentityRejectsMalformedAndOverflowedEnvironment|TestGoCommandQueriesRejectMalformedData)$'],180000);
