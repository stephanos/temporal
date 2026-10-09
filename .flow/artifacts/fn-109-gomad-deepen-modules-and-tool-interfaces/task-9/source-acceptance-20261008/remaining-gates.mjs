import {spawnSync} from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
const out=path.dirname(new URL(import.meta.url).pathname),repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const historical=JSON.parse(fs.readFileSync(path.join(out,'historical-inputs.json'))),control=JSON.parse(fs.readFileSync(path.join(out,'public-control-inputs.json')));
const fixture=`TASK9_REPO=${control.scratch} go run -tags test_dep ${control.scratch}/fixture.go`;
const portable='^Test(ProjectBuildInfo|NormalizeCapabilityMode|PreparationBuildEnvironment|PreparedRecordProjection|ReadProvenanceRoundTrips|ValidateProvenance|ValidateExecCapabilityModules)';
const lint='/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0',errortype='/tmp/fn109-lint-tools.ZdNe1t50/errortype';
const makeInputs=`GOLANGCI_LINT_FIX=false GOLANGCI_LINT=${lint} ERRORTYPE=${errortype}`;
const gates=[
 ['public-payload-buildinfo',`go version -m ${control.payload}`],
 ['archive-cross-graph-cache',`node ${out}/public-control-cache.mjs cross-graph`],
 ['public-prepared-historical-final-fresh',`cd ${historical.final}/tools/gomad3 && ${fixture}`],
 ['archive-historical-final-cache',`node ${out}/public-control-cache.mjs historical-final`],
 ['public-prepared-current-fresh',`cd tools/gomad3 && ${fixture}`],
 ['public-prepared-current-cached',`cd tools/gomad3 && ${fixture}`],
 ...[['original',historical.original],['historical-final',historical.final],['current',repo]].map(([label,graph])=>['download-outcomes-'+label,`cd ${graph}/tools/gomad3 && go run -tags test_dep ${out}/download-outcomes.go`]),
 ...[['original',historical.original],['historical-final',historical.final],['current',repo]].map(([label,graph])=>['original-assertions-'+label,`cd ${graph}/tools/gomad3 && go test -json -count=1 -tags test_dep ./target -run '${portable}'`]),
 ['final-focused',`cd tools/gomad3 && go test -json -count=1 -tags test_dep ./target/... ./internal/hostexec -run 'Test(AdapterSourceSet|ReadToolchainIdentity|GoCommand|PinnedToolchain|Compatibility|Structured|Diagnostic|List|Run|Capture|ClassifyGroupSignal|NewRejects|PreparedCacheDigest|CapabilityReviewGoldenCanonicalBytes|ProjectCapability|ValidateCapability|TargetFileCleanup|ModuleDirectory)'`],
 ...[['darwin','arm64'],['linux','amd64']].map(([goos,goarch])=>['supported-source-'+goos,`cd tools/gomad3 && GOOS=${goos} GOARCH=${goarch} CGO_ENABLED=0 go list -json -tags test_dep ./target/... ./internal/hostexec`]),
 ['final-generated-check',`GOFLAGS=-tags=test_dep make -C tools/gomad3 validate`],
 ['final-lint-fast',`make lint-code-fast ${makeInputs} GOLANGCI_LINT_BASE_REV=29c80199cd`],
 ['final-lint-integrated-original',`make lint-code-gomad3 ${makeInputs} GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025`],
 ['final-format',`gofmt -l tools/gomad3/target/go_command_source_test.go && git diff --check`],
];
const results=[];
for(const [name,command]of gates) {
 console.log(JSON.stringify({started:name,command}));
 const r=spawnSync(process.execPath,[path.join(out,'run.mjs'),name,command],{cwd:repo,stdio:'inherit'});
 results.push({name,exit:r.status,signal:r.signal});
 if(r.signal||r.status==null)break;
}
fs.writeFileSync(path.join(out,'remaining-gates.json'),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(results));
if(results.some(x=>x.exit!==0))process.exitCode=1;
