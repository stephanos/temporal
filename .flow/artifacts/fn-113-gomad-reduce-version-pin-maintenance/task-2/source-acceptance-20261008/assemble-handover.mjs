import fs from 'node:fs';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const directory = '.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008';
const read = name => JSON.parse(fs.readFileSync(`${directory}/${name}`, 'utf8'));
const hash = value => crypto.createHash('sha256').update(value).digest('hex');
const fileHash = name => hash(fs.readFileSync(name));
const command = (program, args) => {
  const result = spawnSync(program, args, {encoding:'utf8'});
  if (result.status !== 0) throw Error(`${program}: ${result.stderr}`);
  return result.stdout.trim();
};
const git = (...args) => command('git', args);
const base = fs.readFileSync('.flow/tmp/base_commit', 'utf8').trim();
const head = git('rev-parse', 'HEAD');
const paths = [...new Set([
  ...git('ls-files', 'tools/gomad3', 'go.mod', 'go.sum', 'tools/gomad3integration/go.mod', 'tools/gomad3integration/go.sum', '.github/.golangci.yml', 'Makefile', 'cmd/tools/lintcode').split('\n'),
  ...git('ls-files', '--others', '--exclude-standard', 'tools/gomad3').split('\n'),
].filter(Boolean))].sort();
const source = paths.map(path => ({path, sha256:fs.existsSync(path) ? fileHash(path) : null}));
const sourceHash = hash(JSON.stringify(source));
const expectedSource = '1ea646fc66ac63aa0d2dde38e434f8b5529c28ac9040b55b7a04c56c4ebc0f3f';
if (sourceHash !== expectedSource) throw Error('final source binding changed');
const acceptedNames = [
  'final-bound-public-packages', 'accepted-cache-tmpfs', 'accepted-deterministicio-workspace-complement',
  'accepted-wire', 'accepted-architecture', 'accepted-retained-generation', 'accepted-retained-list',
  'accepted-retained-assertions', 'accepted-vet', 'final-bound-errortype', 'accepted-darwin-static',
  'accepted-linux-static', 'accepted-validate', 'accepted-lint-unfiltered', 'accepted-version-lint',
  'accepted-lint-fast', 'accepted-release-counterfactual',
];
const receipts = acceptedNames.map(name => {
  const receipt = read(`${name}-receipt.json`);
  const expectedExit = ['accepted-lint-unfiltered', 'accepted-release-counterfactual'].includes(name) ? 1 : 0;
  if (receipt.exit_code !== expectedExit || receipt.source_tree_sha256 !== sourceHash || !receipt.source_unchanged || !receipt.tools_unchanged || !receipt.overlay_unchanged) throw Error(`invalid final receipt: ${name}`);
  if (fileHash(`${directory}/${receipt.log}`) !== receipt.log_sha256) throw Error(`log changed: ${name}`);
  for (const tool of receipt.tools) if (fileHash(tool.path) !== tool.sha256) throw Error(`tool changed: ${tool.path}`);
  for (const entry of receipt.overlay?.entries ?? []) if (fileHash(entry.actual) !== entry.executing_sha256) throw Error(`overlay changed: ${entry.actual}`);
  return receipt;
});
const selection = read('deterministicio-selection.json');
const tmpfsNames = [
  'TestAdapterCacheCleanupOwnerErrors', 'TestAdapterCacheCleanupRegistryPublicationReuseRetry',
  'TestAdapterCacheCleanupValidationControls', 'TestAdapterCacheCleanupRegistryPrimary',
];
const eventNames = name => [...new Set(fs.readFileSync(`${directory}/${name}.log`, 'utf8').split('\n').flatMap(line => {
  try { const event = JSON.parse(line); return event.Test && !event.Test.includes('/') && ['pass','skip'].includes(event.Action) ? [event.Test] : []; } catch { return []; }
}))].sort();
const tmpfsExecuted = eventNames('accepted-cache-tmpfs');
const workspaceExecuted = eventNames('accepted-deterministicio-workspace-complement');
const selected = selection.selected.map(value => value.name).sort();
const union = [...tmpfsExecuted, ...workspaceExecuted].sort();
if (JSON.stringify(union) !== JSON.stringify(selected) || tmpfsExecuted.some(name => workspaceExecuted.includes(name)) || JSON.stringify(tmpfsExecuted) !== JSON.stringify([...tmpfsNames].sort())) throw Error('incomplete complementary selection');
const originalPath = 'tools/gomad3/deterministicio/adapter_cache_cleanup_test.go';
const original = git('show', `${base}:${originalPath}`);
const current = fs.readFileSync(originalPath, 'utf8').trimEnd();
if (original !== current) throw Error('original cleanup controls changed');
const probes = ['cleanup-filesystem-probe', 'cleanup-tmpfs-probe'].map(name => {
  const rows = fs.readFileSync(`${directory}/${name}.log`, 'utf8').trim().split('\n').map(line => JSON.parse(line));
  if (rows.length !== 32 || rows.some(row => row.uid !== 1000 || row.leaf_size_before_denial !== 7 || !row.actual_read_denial.includes('permission denied') || JSON.stringify(row.work_entries_before_remove) !== '["denied"]')) throw Error('invalid actual filesystem probe');
  return {name, receipt:`${name}-receipt.json`, log_sha256:fileHash(`${directory}/${name}.log`), cases:rows.length, permission_cleanup:rows.filter(row => row.cleanup_is_permission).length, nil_cleanup:rows.filter(row => row.cleanup_error === '<nil>').length};
});
const tmpfsDirectory = '/dev/shm/gomad-fn1132-cleanup-3pOD0l3j';
const filesystem = {
  source_tree_sha256:sourceHash,
  original_test_file:originalPath,
  original_and_current_file_sha256:fileHash(originalPath),
  original_body_preserved:true,
  effective_TMPDIR:tmpfsDirectory,
  mount:command('findmnt', ['-T', tmpfsDirectory, '-n', '-o', 'TARGET,SOURCE,FSTYPE,OPTIONS']),
  workspace_mount:command('findmnt', ['-T', process.cwd(), '-n', '-o', 'TARGET,SOURCE,FSTYPE,OPTIONS']),
  current_tmpfs_children:fs.readdirSync(tmpfsDirectory),
  parent_retained_for_conductor:true,
  tools:receipts[0].tools,
  unchanged_build_environment:Object.fromEntries(['GOCACHE','GOTMPDIR','GOMODCACHE'].map(key => [key, receipts[1].environment[key]])),
  probe_program_sha256:fileHash(`${directory}/cleanup_probe.go`),
  probes,
  tmpfs_selected:tmpfsExecuted,
  workspace_selected:workspaceExecuted,
  complete_selected_count:selected.length,
  union_complete:true,
  overlap:[],
  full_single_environment_package_pass:false,
};
const preservation = read('preservation.json');
for (const file of preservation.userFiles) if (fileHash(file.file) !== file.sha256) throw Error('user file changed');
const lint = read('lint-attribution.json');
const bindings = ['assertion-mapping.md','deterministicio-selection.json','preservation.json','lint-attribution.json','private-module-cache.json','retained_overlay.go','retained-assertions-binding.json','retained-assertions-generated.go','retained-assertions-overlay.json','counterfactual.mjs','release-counterfactual-binding.json','transaction-release-discard.go','release-counterfactual-overlay.json','run.mjs','gates.mjs'].map(path => ({path,sha256:fileHash(`${directory}/${path}`)}));
const evidence = {
  task_id:'fn-113-gomad-reduce-version-pin-maintenance.2', status:'in_progress', base_commit:base,
  head_at_handover:head, commits:git('rev-list','--reverse',`${base}..${head}`).split('\n').filter(Boolean), prs:[],
  tests:receipts.map(receipt => receipt.command),
  tier:'Tier: session (jev-unavailable(no_key))',
  review:{mode:'host-deferred', dispatched_by_worker:false, verdict:null, owner:'conductor; commit candidate before independent integrated review'},
  implementation_state:'uncommitted frozen candidate; conductor owns staging/commit, review and lifecycle',
  command_ownership:{worker_commands_terminal:true, go_lane_released:true, conductor_replay_not_counted_as_unique_coverage:true},
  source_tree_sha256:sourceHash, source_scope_manifest:'final-bound-public-packages-source.json',
  source_binding_algorithm:'SHA256(JSON.stringify(sorted complete scoped tracked+untracked path/hash objects)); re-enumerated before/after each final gate',
  current_source_checked_at:new Date().toISOString(),
  host:{platform:process.platform, architecture:process.arch, uid:process.getuid(), patched_driver_present:fs.existsSync('tools/gomad3/.toolchain/bin/go'), native_qualification_claimed:false},
  environment:receipts[0].environment, tools:receipts[0].tools,
  baseline:{validate:'green; baseline-validate.log',parent_quick:'red; baseline-parent-quick.log; no blanket green handoff',scoped_lint:'red; 100 findings, including 14 R3-owned; baseline-lint-receipt.json',public_no_fault:'diagnostic-characterization-before and public-release-before; exact behavior preserved'},
  final_gates:receipts.map(receipt => ({name:receipt.name,receipt:`${receipt.name}-receipt.json`,receipt_sha256:fileHash(`${directory}/${receipt.name}-receipt.json`),command:receipt.command,exit_code:receipt.exit_code,counts:receipt.counts,skips:receipt.skips.map(value => value.Test),source_tree_sha256:receipt.source_tree_sha256,effective_source_tree_sha256:receipt.effective_source_tree_sha256,source_unchanged:receipt.source_unchanged,tools_unchanged:receipt.tools_unchanged,overlay_unchanged:receipt.overlay_unchanged,expected_red:receipt.name === 'accepted-lint-unfiltered' ? '86 exact unchanged OTHER diagnostics; zero owned findings' : receipt.name === 'accepted-release-counterfactual' ? 'injected private helper mutant releases once/discards cause; eight expected assertion failures, not real hostfs/base behavioral RED' : null})),
  lint:{same_command_scope:lint.same_command_scope,baseline:lint.baseline_count,remaining_other:lint.remaining_count,owned_resolved:lint.owned_resolved.length,owned_remaining:0,exact_other_diagnostics_unchanged:lint.exact_other_diagnostics_unchanged,attribution:'lint-attribution.json',added_version_scope:'separate accepted-version-lint; zero findings',mandatory_fast_lint:'accepted-lint-fast exit0; diff-filtered, not global lint green'},
  validation_effective_GOCACHE:`${process.cwd()}/tools/gomad3/.toolchain/generator-cache`,
  preservation:{record:'preservation.json',tail_equivalence_proofs:4,public_signatures_unchanged:true,root_go_mod_sha256:preservation.root_go_mod_sha256,root_go_sum_sha256:preservation.root_go_sum_sha256,selected_roots:preservation.root_selected,absent_roots:preservation.root_absent,moved_roots:preservation.root_moved,pins:preservation.root_pins,user_files:preservation.userFiles,generated_outputs_unchanged:true},
  coverage:{assertion_map:'assertion-mapping.md',name_and_origin_map:'deterministicio-selection.json',selected_names:selection.selected.length,excluded_original_wrappers:selection.excluded.length,filesystem_union:'filesystem-union.json',retained_overlay:{tests:25,origin_bodies:28,admitted_transformations:64,pass_events:70,effective_source_tree_sha256:receipts.find(receipt => receipt.name === 'accepted-retained-assertions').effective_source_tree_sha256},stock_source_pins:30,root_graph_listings:18,no_full_native_or_single_environment_test_host_pass:true},
  preserved_failures:[
    {record:'baseline-parent-quick.log',classification:'initial absent patched-driver/unsupported-host outcomes and stale libc fixture; not pass'},
    {record:'baseline-lint-receipt.json',classification:'actual unchanged-source 14 owned lint RED; no behavioral diagnostic change claimed'},
    {record:'portable-preservation-consumers-receipt.json',classification:'module extraction ENOSPC; not test coverage; private cache repair provenance in private-module-cache.json'},
    {record:'portable-consumers-private-cache-receipt.json',classification:'actual stock nm SIGBUS from root telemetry mmap; repaired by task-private TEST_TELEMETRY_DIR; subsequent actual assertions execute'},
    {record:'final-public-packages-receipt.json',classification:'invalidated mid-gate source edit; source_unchanged=false; no final acceptance credit'},
    {record:'frozen-public-packages-receipt.json',classification:'new deleted-CWD fixture successive Getwd provenance plus real 22 ms GNU make future-mtime warning'},
    {record:'accepted-public-packages-receipt.json',classification:'real 26 ms GNU make warning; exact admitted fixture-only fixed-past Chtimes correction'},
    {record:'accepted-deterministicio-receipt.json',classification:'additive graph fixture transport/inventory errors; final uses actual source-set owner, not duplicated foreign projection'},
    {record:'final-bound-deterministicio-receipt.json',classification:'unchanged workspace cache-cleanup real-fault assertion failure; exact four original controls execute in tmpfs, complementary coverage complete'},
    {record:'accepted-errortype-receipt.json',classification:'wrong flag setup exit2; corrected final-bound-errortype passes'},
    {record:'accepted-release-counterfactual-receipt.json',classification:'injected counterfactual expected RED; not base behavior or real Lock.Release hostfs fault'},
  ],
  remaining_requirements:[
    {requirement:'independent current integrated source review, candidate commit, metadata/lifecycle completion',owner:'conductor',state:'required before source acceptance/done; worker reports no verdict'},
    {requirement:'qualified public Default/profile/capability/Bootstrap/host-pack wrappers; patched runtime and functional/full native test-host, reports/replay/soak',owner:'fn149.2/fn149.4 Darwin; fn128.4/fn128.7 Linux',state:'deferred and unverified; portable counterparts do not qualify native wrappers'},
    {requirement:'TestMemberlistSuppliedTCPConsumer external consumer checkout modfile routing and actual TCP membership lifecycle',owner:'retained platform workload under fn149/fn128; exact supplied checkout absent',state:'not executed; ordinary private stock consumer, source inventories and dual pins executed'},
    {requirement:'actual SDK Git checkout prerequisite',owner:'fn105 D8-D10',state:'module cache is not checkout; not prerequisite or satisfaction claim for R3'},
    {requirement:'global residual lint outside R3-owned source sites',owner:'exact OTHER source sites in lint-attribution.json',state:'86 unchanged findings retained; no suppression or waiver'},
  ],
  artifact_bindings:bindings,
};
console.log(JSON.stringify({files:{'filesystem-union.json':JSON.stringify(filesystem,null,2)+'\n','evidence.json':JSON.stringify(evidence,null,2)+'\n'}}));
