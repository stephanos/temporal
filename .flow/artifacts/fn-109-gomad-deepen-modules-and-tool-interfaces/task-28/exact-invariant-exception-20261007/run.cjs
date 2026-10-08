const fs = require('node:fs');
const path = require('node:path');
const cp = require('node:child_process');
const crypto = require('node:crypto');
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.join(root, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/exact-invariant-exception-20261007');
const scratch = path.join(root, '.flow/tmp/fn10928-exception');
const goDir = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const hash = data => crypto.createHash('sha256').update(data).digest('hex');
const [id, relativeCwd, command, ...args] = process.argv.slice(2);
fs.mkdirSync(out, {recursive:true});
const environment = {GOWORK:'off', GOTOOLCHAIN:'local', GOPROXY:'off', GOSUMDB:'off', GOENV:'off', GOFLAGS:'', PATH:goDir+':'+process.env.PATH, LINT_POLICY_GOLANGCI:'/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0'};
const env = {...process.env, ...environment};
delete env.GOMADSEED;
delete env.GOMAD3_CHILD_SEED;
const bindings = {};
for (const file of ['.github/.golangci.yml','cmd/tools/lintcode/lint_policy_test.go','tools/gomad3/runner/internal/campaign/controller.go','tools/gomad3/runner/internal/campaign/controller_test.go','tools/gomad3/runner/internal/campaign/controller_completion_test.go','tools/gomad3/Makefile','go.mod','go.sum','tools/gomad3/go.mod','tools/gomad3/go.sum']) bindings[file] = hash(fs.readFileSync(path.join(root,file)));
const fixtureBindings = {};
if (relativeCwd !== '.' && relativeCwd !== 'tools/gomad3') {
  for (const file of ['.github/.golangci.yml','cmd/tools/lintcode/lint_policy_test.go','tools/gomad3/runner/internal/campaign/controller.go']) {
    const actual = path.resolve(root,relativeCwd,file);
    if (fs.existsSync(actual)) fixtureBindings[actual] = hash(fs.readFileSync(actual));
  }
}
const logPath = path.join(scratch,id+'.log');
const fd = fs.openSync(logPath,'w');
const start = new Date();
const timer = process.hrtime.bigint();
const result = cp.spawnSync(command,args,{cwd:path.resolve(root,relativeCwd), env, stdio:['ignore',fd,fd], timeout:600000});
const elapsedSeconds = Number(process.hrtime.bigint()-timer)/1e9;
fs.closeSync(fd);
const bytes = fs.readFileSync(logPath);
const storage = JSON.stringify({encoding:'base64', original_sha256:hash(bytes), content:bytes.toString('base64')})+'\n';
const storedPath = path.join(out,id+'.log.json');
fs.writeFileSync(storedPath,storage);
const receipt = {id,cwd:path.resolve(root,relativeCwd),argv:[command,...args],environment,unset:['GOMADSEED','GOMAD3_CHILD_SEED'],started_at:start.toISOString(),ended_at:new Date().toISOString(),elapsed_seconds:elapsedSeconds,exit_code:result.status,signal:result.signal,error:result.error?.message??null,source_sha256:bindings,fixture_sha256:fixtureBindings,log:{path:path.relative(root,storedPath),encoding:'base64',original_sha256:hash(bytes),storage_sha256:hash(storage),original_bytes:bytes.length}};
fs.writeFileSync(path.join(out,id+'.receipt.json'),JSON.stringify(receipt,null,2)+'\n');
console.log(JSON.stringify({id,exit_code:result.status,signal:result.signal,elapsed_seconds:elapsedSeconds,log:logPath}));
process.exitCode = result.status??2;
