import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {readSourceManifest,sourceManifestBinding,sourceManifestIndex} from './source-manifests.mjs';
const repo=process.cwd(),out=path.dirname(new URL(import.meta.url).pathname),base='73a37433b526ae9d6165ffaf1400460f9a6b36d8';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const command=(...args)=>{const r=spawnSync(args[0],args.slice(1),{encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);return r.stdout;};
const read=name=>JSON.parse(fs.readFileSync(path.join(out,name)));
const receipt=name=>read(name+'-receipt.json');
const events=name=>fs.readFileSync(path.join(out,name+'.log'),'utf8').split('\n').filter(x=>x.startsWith('{')).flatMap(x=>{try{return [JSON.parse(x)];}catch{return [];}}).filter(x=>x.Action);
const final=readSourceManifest('final-cli-packs-source.json'), frozen=hash(JSON.stringify(final));
const paths=[...new Set([...command('git','ls-files','tools/gomad3','go.mod','go.sum','tools/gomad3integration/go.mod','tools/gomad3integration/go.sum','.github/.golangci.yml','Makefile','cmd/tools/lintcode').split('\n'),...command('git','ls-files','--others','--exclude-standard','tools/gomad3').split('\n')].filter(Boolean))].sort().map(name=>({path:name,sha256:hash(fs.readFileSync(name))}));
if(hash(JSON.stringify(paths))!==frozen)throw Error('source no longer frozen');
const all=fs.readdirSync(out).filter(x=>x.endsWith('-receipt.json')).sort().map(name=>{
 const r=read(name),log=fs.readFileSync(path.join(out,r.log));
 if(hash(log)!==r.log_sha256||!r.source_unchanged||!r.tools_unchanged)throw Error('invalid receipt '+name);
 const manifest=sourceManifestBinding(r.name+'-source.json');
 if(manifest.source_tree_sha256!==r.source_tree_sha256)throw Error('receipt manifest binding mismatch '+name);
 return {...r,receipt:name,source_manifest:manifest,observed_test_counts:r.command.includes('test ')&&r.command.includes('-json')?r.counts:null,counts_note:r.command.includes('test ')&&!r.command.includes('-json')?'text-output JSON counters unavailable, not zero tests':undefined};
});
if(all.length!==sourceManifestIndex().logical_gate_bindings)throw Error('logical gate manifest inventory differs');
const selectedFinal=['final-serialized-affected','final-pinimpact-portable','final-build-refusal-counterparts','final-architecture','publication-parent-tmpfs'];
for(const name of [...selectedFinal,'final-cli-packs','final-validate','final-darwin-static','final-linux-static','final-vet','final-errortype','final-lint-fast','final-lint-unfiltered','final-format'])if(receipt(name).source_tree_sha256!==frozen)throw Error('wrong candidate '+name);
const mixed=events('final-serialized-affected'), ordinary=events('final-cli-packs'), tmpfs=events('publication-parent-tmpfs');
const rootPackage='go.temporal.io/server/tools/gomad3/upgrade',parent='TestRunReportsPublicationFailureAndKeepsPriorDossier';
const tests=e=>new Map(e.filter(x=>x.Test&&['pass','fail','skip'].includes(x.Action)).map(x=>[x.Package+'::'+x.Test,{package:x.Package,test:x.Test,outcome:x.Action}]));
const mixedTests=tests(mixed),packetTests=new Map(mixedTests),packets=[];
for(const [key,row] of packetTests)if(row.package===rootPackage&&(row.test===parent||row.test.startsWith(parent+'/')))packetTests.delete(key);
for(const [key,row] of tests(tmpfs)){if(row.package!==rootPackage||(row.test!==parent&&!row.test.startsWith(parent+'/'))||row.outcome!=='pass')throw Error('tmpfs selection exceeds original parent');packetTests.set(key,{...row,receipt:'publication-parent-tmpfs-receipt.json'});}
if([...mixedTests.keys()].sort().join('\n')!==[...packetTests.keys()].sort().join('\n'))throw Error('publication filesystem union omitted or added tests');
const ordinaryRoot=[...tests(ordinary)].filter(([,x])=>x.package===rootPackage).map(([key])=>key).sort();
if(ordinaryRoot.join('\n')!==[...packetTests].filter(([,x])=>x.package===rootPackage).map(([key])=>key).sort().join('\n'))throw Error('ordinary root test inventory differs');
for(const row of packetTests.values())if(row.outcome==='fail')throw Error('unproved affected source assertion '+row.test);
const packages=[...new Set(mixed.filter(x=>x.Package).map(x=>x.Package))].sort();
for(const pkg of packages){const rows=[...packetTests.values()].filter(x=>x.package===pkg);if(!rows.length)throw Error('empty package packet '+pkg);if(pkg!==rootPackage&&!mixed.some(x=>x.Package===pkg&&!x.Test&&x.Action==='pass'))throw Error('whole package not passed '+pkg);packets.push({package:pkg,scope:pkg===rootPackage?'exact workspace complement plus four original parent/subtest identities in admitted tmpfs':'whole package actually passes within raw mixed RED command',receipts:pkg===rootPackage?['final-serialized-affected-receipt.json','publication-parent-tmpfs-receipt.json']:['final-serialized-affected-receipt.json'],counts:{pass:rows.filter(x=>x.outcome==='pass').length,skip:rows.filter(x=>x.outcome==='skip').length,fail:0},tests:rows});}
for(const name of ['final-pinimpact-portable','final-build-refusal-counterparts','final-architecture']){
 if(receipt(name).exit_code!==0)throw Error('portable packet not green '+name);
 for(const [key,row] of tests(events(name)))packetTests.set(key,{...row,receipt:name+'-receipt.json'});
}
const rootTests=[...packetTests.values()].filter(x=>x.package===rootPackage),union={source_tree_sha256:frozen,original_mixed_exit:receipt('final-serialized-affected').exit_code,original_mixed_counts:receipt('final-serialized-affected').counts,original_broad_red:'ordinary-r4-packages-receipt.json',single_environment_or_original_broad_pass:false,filesystem_separated_union:true,exact_publication_parent_original_sha256:read('publication-parent-tmpfs-binding.json').original_sha256,required_root_test_set_equal:true,no_omitted_source_test_identities:true,unique_named_test_identities:packetTests.size,counts:{pass:[...packetTests.values()].filter(x=>x.outcome==='pass').length,skip:[...packetTests.values()].filter(x=>x.outcome==='skip').length,fail:[...packetTests.values()].filter(x=>x.outcome==='fail').length},packets,other_packet_receipts:['final-pinimpact-portable-receipt.json','final-build-refusal-counterparts-receipt.json','final-architecture-receipt.json'],publication_workspace_complement:rootTests.filter(x=>x.test!==parent&&!x.test.startsWith(parent+'/')).map(x=>x.test),publication_tmpfs_exact_parent:rootTests.filter(x=>x.test===parent||x.test.startsWith(parent+'/')).map(x=>x.test)};
fs.writeFileSync(path.join(out,'green-packet-union.json'),JSON.stringify(union,null,2)+'\n');
const predecessor=read('../../task-2/source-acceptance-20261008/deterministicio-selection.json');
const wrappers=predecessor.excluded.map(row=>{const file='tools/gomad3/deterministicio/'+row.file,actual=hash(fs.readFileSync(file));if(actual!==row.source_sha256)throw Error('predecessor original owner changed '+file);return {...row,path:file,current_sha256:actual,predecessor_proof:'../../task-2/source-acceptance-20261008/assertion-mapping.md',current_R4_execution_credit:false,native_owners:['fn149.2/fn149.4','fn128.4/fn128.7']};});
const baselineFails=[...fs.readFileSync(path.join(out,'baseline-quick.log'),'utf8').matchAll(/^--- FAIL: (\S+)/gm)].map(x=>x[1]);
for(const name of baselineFails)if(!wrappers.some(x=>x.name===name))throw Error('unmapped baseline original failure '+name);
const pinimpactExcluded=['TestFixtureBumpMatchesBuildRejections','TestSameVersionWithChangedSum','TestReplacedModules'];
const pinimpactFiles=fs.readdirSync('tools/gomad3/upgrade/pinimpact').filter(x=>x.endsWith('_test.go'));
const pinimpactAll=pinimpactFiles.flatMap(file=>[...fs.readFileSync('tools/gomad3/upgrade/pinimpact/'+file,'utf8').matchAll(/^func (Test\w+)\(/gm)].map(x=>x[1])).sort();
const pinimpactRun=events('final-pinimpact-portable').filter(x=>x.Action==='run'&&x.Test&&!x.Test.includes('/')).map(x=>x.Test).sort();
if(pinimpactAll.filter(x=>!pinimpactExcluded.includes(x)).join('\n')!==pinimpactRun.join('\n'))throw Error('pinimpact ordinary complement incomplete');
fs.writeFileSync(path.join(out,'original-owner-mapping.json'),JSON.stringify({baseline_failed_original_names:baselineFails,deterministicio_unsupported_originals:wrappers,pinimpact_all_original_top_levels:pinimpactAll,pinimpact_excluded_qualified_wrappers:pinimpactExcluded,pinimpact_exact_ordinary_complement:true,pinimpact_source_counterparts:['TestPortableFixturePinDecisions','TestAdapterRegistryPortablePinDecisions','TestPortableAdapterConfigurationRefusals'],native_qualification_unverified:true},null,2)+'\n');
const artifactOnly=process.argv.includes('--artifact-only');
const task=artifactOnly?{status:read('evidence.json').status}:JSON.parse(command('/home/agent/.codex/scripts/flowctl','show','fn-113-gomad-reduce-version-pin-maintenance.3','--json'));
if(task.status!=='in_progress')throw Error('task lifecycle unexpectedly changed '+task.status);
const head=command('git','rev-parse','HEAD').trim(),headPaths=command('git','diff','--name-only',base+'..HEAD').trim().split('\n').filter(Boolean);
if(headPaths.some(x=>!x.startsWith('.flow/')&&x!=='MILESTONES.md'))throw Error('nonmetadata committed range changed');
const preservation=read('preservation.json'),lint=read('lint-attribution.json');
const requiredNames=['TestRunCompatibilityPackRefreshResolvesTwoModulesAndKeepsPartialApproval','TestRunCompatibilityPackRefreshContinuesAfterActualDiscoveryFailure','TestRunCompatibilityPackRefreshDiagnosticsPreservePrimaryStatus','TestRefreshReportsOtherPlatformRequestsAndLeavesThemUntouched','TestRefreshNeverTreatsAnApprovalOfOlderEvidenceAsCurrent','TestSelectedLibcVariantsResolveTheirMappedModuleVersions'];
const assertions=requiredNames.map(name=>{const rows=[...packetTests.values()].filter(x=>x.test===name||x.test.startsWith(name+'/'));if(!rows.length||rows.some(x=>x.outcome!=='pass'))throw Error('missing retained R4 assertions '+name);return {name,receipt:'final-serialized-affected-receipt.json',executed_test_identities:rows};});
const actualInputOutputs=mixed.filter(x=>x.Output&&(/proxy .* zip |stock Go .* actual candidate|mapped modernc-libc-xsys|healthy status|diagnostic write executed/.test(x.Output))).map(x=>({package:x.Package,test:x.Test,output:x.Output.trim()}));
fs.writeFileSync(path.join(out,'r4-named-assertions.json'),JSON.stringify({source_tree_sha256:frozen,assertions,actual_input_outputs:actualInputOutputs,private_callback_stock_closure_not_production_qualified_wrapper:true},null,2)+'\n');
const evidence={task_id:'fn-113-gomad-reduce-version-pin-maintenance.3',status:'in_progress',source_candidate_uncommitted:true,commits:[],base_commit:base,existing_head:head,existing_metadata_only_commits_since_base:command('git','rev-list','--reverse',base+'..HEAD').trim().split('\n').filter(Boolean),tests:all.map(x=>x.command),prs:[],tier:'session (jev-unavailable(no_key))',review:'host-deferred; root owns independent review and verdict',baseline:{quick:'red before edits; inherited unsupported original wrappers; text test counters unavailable',validate:'exit0',lint:'exit1;86 findings,8 owned and78 OTHER',source_tree_sha256:'1ea646fc66ac63aa0d2dde38e434f8b5529c28ac9040b55b7a04c56c4ebc0f3f'},final_source_tree_sha256:frozen,final_source_paths:final.length,green_packet_union:'green-packet-union.json',named_requirements:'r4-named-assertions.json',assertion_mapping:'assertion-mapping.md',original_owner_mapping:'original-owner-mapping.json',preservation:'preservation.json',lint_attribution:'lint-attribution.json',owned_lint_findings:0,other_lint_findings:lint.remaining_count,no_global_lint_green:true,cleanup_failure_diagnosis:'cleanup-diagnosis.json',publication_diagnostic_overlay:'upgrade-publication-overlay-binding.json',publication_budget:'upgrade-parent-fixture-budget.json',exact_parent_filesystem_binding:'publication-parent-tmpfs-binding.json',environment_note:'Raw receipt.environment is harness base; command-local TMPDIR override and binding.effective_environment supply actual original-parent child environment. Raw receipt alone is not effective-override proof.',raw_broad_and_serialized_gates_remain_red:true,root_cause_unknown:true,no_cleanup_or_production_test_fix:true,no_native_qualification_or_full_native_test_host_claim:true,native_owners:['fn149.2/fn149.4','fn128.4/fn128.7'],external_checkout_prerequisites_untouched:true,all_worker_command_handles_terminal:true,receipts:all,preservation_script_sha256:hash(fs.readFileSync(path.join(out,'preservation.mjs'))),assembler_sha256:hash(fs.readFileSync(new URL(import.meta.url)))};
evidence.source_manifest_index={path:'source-manifest-index.json',sha256:hash(fs.readFileSync(path.join(out,'source-manifest-index.json'))),logical_gate_bindings:sourceManifestIndex().logical_gate_bindings,unique_exact_byte_manifests:sourceManifestIndex().unique_exact_byte_manifests};
evidence.manifest_compaction='manifest-compaction.json';
evidence.historical_instrumenter={path:'instrument-publication-executed.mjs',sha256:hash(fs.readFileSync(path.join(out,'instrument-publication-executed.mjs'))),actual_execution_binding:'upgrade-publication-overlay-binding.json',updated_pointer_aware_driver:'instrument-publication.mjs',updated_driver_not_executed:true};
if(evidence.historical_instrumenter.sha256!==read('upgrade-publication-overlay-binding.json').instrumenter_sha256)throw Error('historical diagnostic instrumenter provenance differs');
evidence.artifact_only_assembly=artifactOnly;
if(artifactOnly)evidence.lifecycle_observation='Prior handover in_progress retained; artifact-only assembly performs no Flow read or lifecycle mutation.';
fs.writeFileSync(path.join(out,'evidence.json'),JSON.stringify(evidence,null,2)+'\n');
const summary=`# R4 retained source handover

Refresh now checks its eight existing stderr results while preserving every primary status, diagnostic argument/order and all six stdout checks. Additive real-resolution controls cover two distinct module versions, bad mappings before writes, partial approval/rerun, older-approval refusal, actual discovery failure/continuation and both retained variant selectors. The source candidate is ready for conductor audit, commit and independent review; task remains in_progress and no worker review verdict is claimed.

Base ${base}; HEAD ${head} contains admission metadata only. Worker commits: none. The frozen ${final.length}-path source binding is ${frozen}. [evidence.json](evidence.json) retains commands/exits/log/tool/config/source hashes and all setup/intermediate failures.

[source-manifest-index.json](source-manifest-index.json) maps all32 original gate manifest names to nine retained exact-byte canonical payloads. The23 removed duplicate files are recoverable byte-exact through those payloads plus original SHA/size bindings; [manifest-compaction.json](manifest-compaction.json) binds checked pre-deletion equality, unchanged raw receipts/logs, original gate harness and packet metadata. Embedded evidence records carry explicit canonical pointers. No source, test, tool or staged-index changes accompany compaction.

[instrument-publication-executed.mjs](instrument-publication-executed.mjs) preserves the exact executed diagnostic driver bound by its historical SHA. Its pointer-aware successor and artifact-only inspection/assembly scripts consume the compact index; the successor has not been executed and supplies no new diagnostic credit. The actual run.mjs gate harness remains byte-exact.

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Actual current source packets

[green-packet-union.json](green-packet-union.json) binds ${union.counts.pass} unique parent-inclusive passing test identities, ${union.counts.skip} inherited host-profile skip and zero unresolved selected source failures. This is an explicit filesystem-separated union, not a passing broad command or single-environment full package/native gate. Each original publication-parent identity is included exactly once, and all other required package identities remain selected.

- Full CLI, adapterregen, version and pack/authoring packages actually pass within [final-serialized-affected](final-serialized-affected-receipt.json), whose overall exit1 remains RED (682 pass/2 fail/1 skip). The exact original default pipeline passes and publishes six staged-digest-matched files there.
- Upgrade root's workspace complement plus the unchanged exact publication-failure parent in [publication-parent-tmpfs](publication-parent-tmpfs-receipt.json) covers the complete original root test inventory. Its three subcases and parent pass4 events without skip; original source/assertions/cleanup are unchanged.
- Exact pinimpact ordinary complement passes121 events; only three unchanged qualified BuildAdapter wrappers are excluded and mapped to actual report/pack-selection plus registry/preparation source counterparts (27 passing events). [original-owner-mapping.json](original-owner-mapping.json) binds original names/current hashes and retained native owners.
- Full architecture suite passes78 events. Exact check-only validate, both darwin/arm64 and linux/amd64 SOURCE list/vet sets, scoped vet, errortype with style-check=false, formatting and mandatory fixes-disabled make lint-code-fast exit0. Unfiltered same-scope lint remains exit1:86→78, all eight owned sites resolved, zero owned new findings, all78 OTHER full blocks and owner bytes identical. [lint-attribution.json](lint-attribution.json) gives exact owners; no global lint-green claim follows.

## Real inputs and preservation

[r4-named-assertions.json](r4-named-assertions.json) binds executed names and actual ZIP h1 checksums, selected stock Go/root, module versions/sums, source hashes and fresh approval digests. The private existing callback executes actual stock target.ReviewCapabilities and authoring discovery, not the production qualified reviewer wrapper or native workloads. [assertion-mapping.md](assertion-mapping.md) maps every retained criterion, error and unsupported original.

[preservation.json](preservation.json) reconstructs the complete production file exactly after removing only eight new check wrappers. It binds1,027 unchanged original source/config paths, three additive test files, all six v041 files byte-exact to56148912 (approval and report EOF blank line included), current v047/generated/mutation/inventory controls and root pins/sums/guards. Actual read-only mappings still select x/sys v0.41.0 and v0.47.0; no variant or selecting fixture is removed. This proves policy selection, not native closure availability. Both user files retain their dispatched hashes and untracked/unstaged status; actual SDK checkout is untouched.

## Retained failures and environment limits

The baseline exact Quick is RED on inherited unsupported originals; its text-output JSON counters are unavailable, not zero execution. Initial local-proxy cleanup/expected-message fixture failures, missing-Go characterization setup mismatch and four deprecated-GOROOT lint findings remain retained as setup/intermediate observations, not behavioral RED. Unchanged-source stderr characterization passes10 records (nine subtests plus parent, including extra missing-request-directory load case); the actual eight owned lint findings supply the separate RED.

The original ordinary broad run retains its real default-pipeline ENOTEMPTY verifier-cleanup failure. The admitted serialized gate retains a different original portable ENOENT prior-dossier preservation failure. Bounded no-child/listing/actual-verifier probes and the one admitted instrumentation diagnostic do not reproduce those failures; cause and repair remain unknown. [cleanup-diagnosis.json](cleanup-diagnosis.json), [upgrade-publication-overlay-binding.json](upgrade-publication-overlay-binding.json) and original logs remain intact. Instrumentation preserves the entire original body/fixture calls/assertions; its four passes are diagnostic-only, not acceptance.

Root separately admitted only the unchanged publication parent in unique private0700 rw/noexec tmpfs scratch. [publication-parent-tmpfs-binding.json](publication-parent-tmpfs-binding.json) binds the command-local effective TMPDIR, actual mount/capacity, original source, exact selection and checked empty/removed parent. Raw receipt.environment records the harness base TMPDIR; command assignment and binding.effective_environment record the child override. GOTMPDIR/build/tool/module caches and all executable paths remain workspace/absolute /bin/sh. The original permission-class and prior-byte assertions execute, but no printed numeric Run errno or syscall trace is claimed. Dynamic peak footprint was not captured; the source-derived conservative64KiB budget is explicitly distinguished from a measured peak. Two empty diagnostic probe parents were checked removed; the original failed verifier scratch is preserved for investigation. No cleanup retry/fix, broad relocation, pin/guard widening, production fault seam or native transfer resolves either raw RED.

## Conductor and native owners

Root still owns nonempty source commit, fresh independent gpt-6.1-sol/high review, routed fixes and Flow completion. Native Darwin fn149.2/fn149.4 and Linux fn128.4/fn128.7 qualified wrappers/current native discovery/workloads/replay/full test-host/soak remain deferred and unverified. R3's verified unchanged source counterparts/origin overlay are referenced as predecessor proof, never fresh R4 original/native execution credit. Supplied Memberlist TCP checkout and actual SDK checkout prerequisites remain unclaimed. No PR/push/CI authority or review/Done follows from this handover.

All worker-started command handles are terminal; the sole Go lane is released. Candidate remains uncommitted and task in_progress.
`;
fs.writeFileSync(path.join(out,'handover.md'),summary);
console.log(JSON.stringify({status:task.status,frozen,unique_named_tests:union.unique_named_test_identities,counts:union.counts,packets:packets.map(x=>({package:x.package,counts:x.counts})),handover:path.join(out,'handover.md'),evidence:path.join(out,'evidence.json')}));
