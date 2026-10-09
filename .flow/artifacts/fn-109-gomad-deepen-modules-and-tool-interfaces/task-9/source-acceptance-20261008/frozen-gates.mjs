import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const prior=name=>JSON.parse(fs.readFileSync(path.join(out,name+'-receipt.json'))).command;
const final=JSON.parse(fs.readFileSync(path.join(out,'final-control-inputs.json'))),listing=JSON.parse(fs.readFileSync(path.join(out,'listing-control-v2-inputs.json'))),standard=JSON.parse(fs.readFileSync(path.join(out,'standard-stderr-inputs.json'))),publicControl=JSON.parse(fs.readFileSync(path.join(out,'public-control-inputs.json')));
const gates=[
 ['listing-current',`cd ${final.current}/tools/gomad3 && TASK9_LISTING_CONTROL=${listing.scratch} go test -json -count=1 -tags test_dep ./target -run '^TestTask9ListingCharacterization$'`],
 ['standard-stderr-current',`cd ${final.current}/tools/gomad3 && TASK9_STANDARD_STDERR_CONTROL=${standard.scratch} go test -json -count=1 -tags test_dep ./target -run '^TestTask9StandardStderrCharacterization$'`],
 ['public-cache-archive',`node ${out}/public-control-cache.mjs frozen-current`],
 ...['fresh','cached'].map(mode=>['public-'+mode,`cd tools/gomad3 && TASK9_REPO=${publicControl.scratch} go run -tags test_dep ${publicControl.scratch}/fixture.go`]),
 ['download',prior('repaired-download-current')],['query',prior('repaired-query-fields-current')],['build',prior('build-outcomes-current-before-fix')],
 ['selection',prior('source-selection-current')],
 ['focused',prior('isolated-candidate-focused')],
 ['command-defaults','cd tools/gomad3 && go test -json -count=1 -tags test_dep ./target/internal/gocommand ./internal/hostexec'],
 ['task8-composition',"cd tools/gomad3 && go test -json -count=1 -tags test_dep ./target -run '^TestCapabilitySource'"],
 ['architecture',prior('final-architecture')],['source-darwin',prior('supported-source-darwin')],['source-linux',prior('supported-source-linux')],
 ['errortype',prior('final-errortype')],['generated',prior('final-generated-check')],
 ...['conformance','toolchain','resolver','adapterregen','upgrade','qualification'].map(name=>['consumer-'+name,prior('source-consumer-'+name)]),
 ['lint-fast',prior('final-lint-fast')],['lint-scoped',prior('lint-current-scoped')],['lint-integrated',prior('final-lint-integrated-original')],
 ['format',`bash ${out}/format-gate.sh gofmt -l tools/gomad3/target/target.go tools/gomad3/target/capability_collection.go tools/gomad3/target/go_command_source_test.go tools/gomad3/target/internal/capabilityreview/list.go tools/gomad3/target/internal/capabilityreview/list_source_test.go tools/gomad3/target/internal/gocommand/*.go tools/gomad3/internal/hostexec/command.go tools/gomad3/internal/hostexec/command_unix.go && git diff --check`],
];
const results=[];
for(const [name,command]of gates){console.log(JSON.stringify({started:'frozen-'+name}));const result=spawnSync(process.execPath,[path.join(out,'run.mjs'),'frozen-'+name,command],{cwd:repo,stdio:'inherit'});results.push({name:'frozen-'+name,exit:result.status,signal:result.signal});if(result.signal||result.status===null)break;}
fs.writeFileSync(path.join(out,'frozen-gates.json'),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
if(results.some(row=>row.exit!==0))process.exitCode=1;
