import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const out=path.dirname(new URL(import.meta.url).pathname),repo='/Users/stephan/Workspace/skunkworks/gomad/temporal';
const version=process.argv[2]??'';
const config=JSON.parse(fs.readFileSync(path.join(out,version?'listing-control-v2-inputs.json':'listing-control-inputs.json'))),head=JSON.parse(fs.readFileSync(path.join(out,'head-baseline-inputs.json')));
const gates=[
 ...[['original',config.original],['historical-final',config.final],['current',config.current]].map(([name,graph])=>['listing-boundary-'+name,`cd ${graph}/tools/gomad3 && TASK9_LISTING_CONTROL=${config.scratch} go test -json -count=1 -tags test_dep ./target -run '^TestTask9ListingCharacterization$'`]),
 ...(!version?[['upgrade-head-baseline',`cd ${head.scratch}/tools/gomad3 && go test -json -count=1 -tags test_dep ./upgrade -run '^TestRunReportsPublicationFailureAndKeepsPriorDossier$'`],
 ['adapter-matched-topology','task9_adapter_root=$(mktemp -d /tmp/task9-adapter-probe.XXXXXX) && cd tools/gomad3 && go run -tags test_dep ../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/adapter-command-gap-2026-10-05/base-process-probe/probe.go "$task9_adapter_root"']]:[]),
];
const results=[];
for(const [name,command]of gates){const r=spawnSync(process.execPath,[path.join(out,'run.mjs'),version+name,command],{cwd:repo,stdio:'inherit'});results.push({name:version+name,exit:r.status,signal:r.signal});if(r.signal||r.status==null)break;}
fs.writeFileSync(path.join(out,version+'listing-gates.json'),JSON.stringify(results,null,2)+'\n',{flag:'wx'});
if(results.some(x=>x.exit!==0))process.exitCode=1;
