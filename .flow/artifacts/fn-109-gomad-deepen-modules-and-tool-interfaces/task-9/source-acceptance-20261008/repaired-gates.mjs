import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const prior=name=>JSON.parse(fs.readFileSync(path.join(out,name+'-receipt.json'))).command;
const config=JSON.parse(fs.readFileSync(path.join(out,'public-control-inputs.json')));
const fixture=`cd tools/gomad3 && TASK9_REPO=${config.scratch} go run -tags test_dep ${config.scratch}/fixture.go`;
const history=JSON.parse(fs.readFileSync(path.join(out,'historical-inputs.json')));
const prefix=process.argv[2]??'';
const gates=[
 ...(prefix?[['matched-public-original',`cd ${history.original}/tools/gomad3 && TASK9_REPO=${config.scratch} go run -tags test_dep ${config.scratch}/fixture.go`],
 ['matched-cache-original',`node ${out}/public-control-cache.mjs telemetry-original`],
 ['matched-public-historical-final',`cd ${history.final}/tools/gomad3 && TASK9_REPO=${config.scratch} go run -tags test_dep ${config.scratch}/fixture.go`],
 ['matched-cache-historical-final',`node ${out}/public-control-cache.mjs telemetry-historical-final`],
 ...[['original',history.original],['historical-final',history.final]].flatMap(([label,graph])=>[
 ['matched-download-'+label,`cd ${graph}/tools/gomad3 && go run -tags test_dep ${out}/download-outcomes.go`],
 ['matched-query-'+label,`cd ${graph}/tools/gomad3 && go run -tags test_dep ${out}/query-outcomes.go`],
 ['matched-build-'+label,`cd ${graph}/tools/gomad3 && TASK9_FAILURE_CONTROL=/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-build-outcomes-T7rId4/control.json go run -tags test_dep ${out}/build-outcomes.go`]])]:[['candidate-public-cache-archive',`node ${out}/public-control-cache.mjs repaired-current`]]),
 ['candidate-public-fresh',fixture],['candidate-public-cached',fixture],
 ['candidate-download',prior('repaired-download-current')],
 ['candidate-query',prior('repaired-query-fields-current')],
 ['candidate-build',prior('build-outcomes-current-before-fix')],
 ['candidate-adapter',prior('adapter-probe-current')],
 ['candidate-selection',prior('source-selection-current')],
 ['candidate-focused',prior('final-focused').replace('Test(AdapterSourceSet','Test(ProjectBuildInfo|NormalizeCapabilityMode|PreparationBuildEnvironment|PreparedRecordProjection|ReadProvenanceRoundTrips|ValidateProvenance|ValidateExecCapabilityModules|AdapterSourceSet')],
 ['candidate-command-defaults','cd tools/gomad3 && go test -json -count=1 -tags test_dep ./target/internal/gocommand ./internal/hostexec'],
 ['candidate-architecture',prior('final-architecture')],
 ['candidate-source-darwin',prior('supported-source-darwin')],
 ['candidate-source-linux',prior('supported-source-linux')],
 ['candidate-errortype',prior('final-errortype')],
 ['candidate-generated',prior('final-generated-check')],
 ['candidate-lint-fast',prior('final-lint-fast')],
 ['candidate-lint-scoped',prior('lint-current-scoped')],
 ['candidate-lint-integrated',prior('final-lint-integrated-original')],
 ['candidate-format','test -z "$(gofmt -l tools/gomad3/target/target.go tools/gomad3/target/go_command_source_test.go tools/gomad3/target/internal/gocommand/*.go tools/gomad3/internal/hostexec/command.go tools/gomad3/internal/hostexec/command_unix.go)" && git diff --check'],
];
const results=[];
for(const [name,command]of gates) {
 console.log(JSON.stringify({started:name,command}));
 const r=spawnSync(process.execPath,[path.join(out,'run.mjs'),prefix+name,command],{cwd:repo,stdio:'inherit'});
 results.push({name:prefix+name,exit:r.status,signal:r.signal});
 if(r.signal||r.status==null)break;
}
fs.writeFileSync(path.join(out,prefix+'repaired-gates.json'),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(results));
if(results.some(x=>x.exit!==0))process.exitCode=1;
