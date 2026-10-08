import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = process.cwd();
const outRoot = path.join(repo, '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/source-acceptance-20261008');
const out = process.env.FN1131_EVIDENCE_VARIANT ? path.join(outRoot, process.env.FN1131_EVIDENCE_VARIANT) : outRoot;
fs.mkdirSync(out,{recursive:true});
const goBin = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const env = {...process.env, PATH: goBin + ':' + process.env.PATH, GOENV:'off', GOWORK:'off', GOTOOLCHAIN:'local', GOCACHE:'/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r', TMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX', GOTMPDIR:'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX'};
const base = fs.readFileSync('.flow/tmp/base_commit','utf8').trim();
const digest = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = p => fs.readFileSync(p);
const git = (...args) => {
  const r=spawnSync('git',args,{cwd:repo,encoding:'utf8'});
  if(r.status!==0) throw Error(r.stderr);
  return r.stdout;
};
const quote = s => "'" + s.replaceAll("'", "'\\''") + "'";
const added = ['tools/gomad3/deterministicio/adapter_pin_decisions_test.go','tools/gomad3/upgrade/pinimpact/portable_fixtures_test.go','tools/gomad3/cmd/gomadtool/pin_impact_diagnostics_test.go'];
const sourcePaths = [...new Set([...git('ls-files','tools/gomad3','go.mod','go.sum','tools/gomad3integration/go.mod','tools/gomad3integration/go.sum').trim().split('\n'),...added])].sort();
const freeze = () => digest(sourcePaths.map(p=>p+'\0'+digest(read(p))).join('\n'));
const frozen = freeze();
const receipts = [];
function run(name, command, expected=0, extra={}) {
  const log=path.join(out,name+'.log'), fd=fs.openSync(log,'w'), start=Date.now();
  const r=spawnSync('timeout',['600','bash','-c',command],{cwd:repo,env:{...env,...extra},stdio:['ignore',fd,fd]});
  fs.closeSync(fd);
  if(freeze()!==frozen) throw Error('source changed during '+name);
  const receipt={name,command,exit_code:r.status,signal:r.signal,error:r.error?.message??null,elapsed_seconds:(Date.now()-start)/1000,log:path.relative(out,log),log_sha256:digest(read(log)),source_tree_sha256:frozen,expected_exit:expected};
  receipts.push(receipt);
  fs.writeFileSync(path.join(out,process.argv[2]+'-commands.json'),JSON.stringify({base_commit:base,environment:env,commands:receipts},(k,v)=>k==='environment'?Object.fromEntries(Object.entries(v).filter(([k])=>['PATH','GOENV','GOWORK','GOTOOLCHAIN','GOCACHE','TMPDIR','GOTMPDIR'].includes(k))):v,2)+'\n');
  console.log(name+' exit='+r.status+' elapsed='+receipt.elapsed_seconds+'s');
  if(expected!==null && r.status!==expected) throw Error(name+' unexpected status; read '+log);
  return receipt;
}
function overlay(name, filename, from, to) {
  const scratch=fs.mkdtempSync(path.join(env.TMPDIR,'fn1131-'+name+'-'));
  const original=read(filename).toString();
  if(original.split(from).length!==2) throw Error('nonunique mutation '+name);
  const replacement=path.join(scratch,path.basename(filename));
  fs.writeFileSync(replacement,original.replace(from,to));
  const overlayPath=path.join(scratch,'overlay.json');
  fs.writeFileSync(overlayPath,JSON.stringify({Replace:{[path.join(repo,filename)]:replacement}}));
  return {overlayPath,source:filename,source_sha256:digest(original),mutated_sha256:digest(read(replacement)),from,to};
}

if(process.argv[2]==='controls') {
  const cases = [
    ['report-adapter', 'tools/gomad3/upgrade/pinimpact/pinimpact.go', 'pin.Status, pin.Reason = StatusInvalidated, match.reason\n\t\t}\n\t\tevaluation.record(pin)\n\t}\n}\n\n// evaluatePacks', 'pin.Status, pin.Reason = StatusUnaffected, match.reason\n\t\t}\n\t\tevaluation.record(pin)\n\t}\n}\n\n// evaluatePacks', './upgrade/pinimpact', '^TestPortableFixturePinDecisions$'],
    ['registry-version','tools/gomad3/deterministicio/adapter_registry.go','if version != definition.identity.Version {','if false && version != definition.identity.Version {','./deterministicio','^TestAdapterRegistryPortablePinDecisions$'],
    ['registry-replacement','tools/gomad3/deterministicio/adapter_registry.go','if replacement.Old.Path == module {','if false && replacement.Old.Path == module {','./deterministicio','^TestAdapterRegistryPortablePinDecisions$'],
    ['registry-sum','tools/gomad3/deterministicio/adapter_registry.go','if found || fields[2] != identity.Sum {','if found {','./deterministicio','^(TestAdapterRegistryPortablePinDecisions|TestAdapterModuleSumRecords)$'],
    ['pack-replacement','tools/gomad3/internal/compatibilitypack/v2_selection.go','return !actual.Replaced && !actual.LocalReplacement && actual.Adapter == nil','return !actual.LocalReplacement && actual.Adapter == nil','./upgrade/pinimpact','^TestPortableFixturePinDecisions$'],
  ];
  const mutations=[];
  for(const [name,file,from,to,pkg,selector] of cases) {
    const mutated=overlay(name,file,from,to);
    mutations.push({...mutated,name,receipt:run('counterfactual-'+name,'go -C tools/gomad3 test -tags test_dep -count=1 -json -overlay='+quote(mutated.overlayPath)+' -run '+quote(selector)+' '+pkg,1)});
  }
  fs.writeFileSync(path.join(out,'counterfactuals.json'),JSON.stringify(mutations, null, 2)+'\n');
} else if(process.argv[2]==='gates') {
  const tests=git('grep','-h','-E','^func Test[A-Za-z0-9_]+\\(','--','tools/gomad3/upgrade/pinimpact/*.go').match(/func (Test\w+)/g).map(s=>s.slice(5));
  const selection=[...tests,'TestPortableFixturePinDecisions'].filter(s=>!['TestFixtureBumpMatchesBuildRejections','TestSameVersionWithChangedSum','TestReplacedModules'].includes(s));
  run('portable-pinimpact','go -C tools/gomad3 test -tags test_dep -count=1 -json -run '+quote('^('+selection.join('|')+')$')+' ./upgrade/pinimpact');
  run('portable-registry','go -C tools/gomad3 test -tags test_dep -count=1 -json -run '+quote('^(TestAdapterRegistryPortablePinDecisions|TestAdapterModuleSumRecords)$')+' ./deterministicio');
  run('cli-and-upgrade-and-packs','go -C tools/gomad3 test -tags test_dep -count=1 -json ./cmd/gomadtool ./upgrade ./internal/compatibilitypack/...');
  run('validate','make -C tools/gomad3 validate');
  run('architecture','go -C tools/gomad3 test -tags test_dep -count=1 -json -run '+quote('^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestExactModuleEdges)$')+' .');
  const scope='./upgrade/... ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...';
  run('vet','go -C tools/gomad3 vet -tags test_dep '+scope);
  run('errortype','go -C tools/gomad3 vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype -style-check=false '+scope);
  const lint='cd tools/gomad3 && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --fix=false --build-tags=test_dep --timeout=10m --config=../../.github/.golangci.yml --path-mode=abs --max-issues-per-linter=0 --max-same-issues=0 '+scope;
  const scratch=fs.mkdtempSync(path.join(env.TMPDIR,'fn1131-lint-base-'));
  const Replace={};
  const baseProduction='tools/gomad3/cmd/gomadtool/pin_impact.go';
  const baseProductionPath=path.join(scratch,'pin_impact.go');
  fs.writeFileSync(baseProductionPath,git('show',base+':'+baseProduction));
  Replace[path.join(repo,baseProduction)]=baseProductionPath;
  for(const file of added) {
    const replacement=path.join(scratch,path.basename(file));
    fs.writeFileSync(replacement,'package '+(file.includes('/deterministicio/')?'deterministicio':file.includes('/cmd/gomadtool/')?'main':'pinimpact_test')+'\n');
    Replace[path.join(repo,file)]=replacement;
  }
  const overlayPath=path.join(scratch,'overlay.json'); fs.writeFileSync(overlayPath,JSON.stringify({Replace}));
  run('lint-base',lint,null,{GOFLAGS:'-overlay='+overlayPath});
  run('lint-final',lint,null);
  run('lint-fast','make lint-code-fast GOLANGCI_LINT_BASE_REV='+base+' GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype ALL_TEST_TAGS=test_dep');
  run('format','test -z "$(gofmt -l '+[...added,baseProduction].join(' ')+')"');
  run('diff-check','git diff --check');
} else if(process.argv[2]==='static') {
  for(const platform of ['darwin/arm64','linux/amd64','linux/arm64']) {
    const [GOOS,GOARCH]=platform.split('/');
    run('static-'+GOOS+'-'+GOARCH,'go -C tools/gomad3 list -tags test_dep -json ./upgrade/pinimpact ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...',0,{GOOS,GOARCH,CGO_ENABLED:'0'});
    run('vet-'+GOOS+'-'+GOARCH,'go -C tools/gomad3 vet -tags test_dep ./upgrade/pinimpact ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...',0,{GOOS,GOARCH,CGO_ENABLED:'0'});
  }
} else if(process.argv[2]==='baseline-lint') {
  const scratch=fs.mkdtempSync(path.join(env.TMPDIR,'fn1131-base-lint-'));
  const worktree=path.join(scratch,'worktree');
  git('worktree','add','--detach',worktree,base);
  const lintCache=path.join(scratch,'lint-cache');fs.mkdirSync(lintCache);
  const scope='./upgrade/... ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/...';
  const command='cd '+quote(path.join(worktree,'tools/gomad3'))+' && /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --fix=false --build-tags=test_dep --timeout=10m --config=../../.github/.golangci.yml --path-mode=abs --max-issues-per-linter=0 --max-same-issues=0 '+scope;
  const receipt=run('lint-base-clean-worktree',command,1,{GOLANGCI_LINT_CACHE:lintCache});
  fs.writeFileSync(path.join(out,'baseline-lint-source.json'),JSON.stringify({base_commit:base,worktree,production_source_sha256:digest(read(path.join(worktree,'tools/gomad3/cmd/gomadtool/pin_impact.go'))),lint_cache:lintCache,receipt},null,2)+'\n');
  git('worktree','remove',worktree);
} else throw Error('expected controls, gates, static or baseline-lint');
