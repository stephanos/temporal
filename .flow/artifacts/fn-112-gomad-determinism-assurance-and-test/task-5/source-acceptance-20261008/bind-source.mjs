import {readFileSync, writeFileSync, statSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = dirname(fileURLToPath(import.meta.url));
const retained = '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/source-acceptance-20261007';
const retainedRef = '1deced3efa4e7000163cb269e70e35f0c6b7dbd7';
const compactRef = 'a936b597b4c62fa50f11a6c16c91111cd52b1ec3';
const sha = data => createHash('sha256').update(data).digest('hex');
const read = path => readFileSync(root + '/' + path);
const json = path => JSON.parse(read(path));
function git(args) {
  const r = spawnSync('git', args, {cwd: root, maxBuffer: 32 << 20});
  assert.equal(r.status, 0, r.stderr.toString());
  return r.stdout;
}
function bound(path, expected) {
  const data = read(path);
  assert.equal(sha(data), expected, 'changed bound input ' + path);
  return {path, bytes: data.length, sha256: expected};
}
const head = git(['rev-parse', 'HEAD']).toString().trim();
const base = readFileSync(out + '/base_commit', 'utf8').trim();
assert.equal(head, base, 'conductor changed HEAD during worker evidence capture');
const raw = json(retained + '/raw-manifest.json');
let rawBytes = 0;
for (const entry of raw) {
  const stored = read(retained + '/raw/' + (entry.retained_name ?? entry.name));
  if (entry.retained_name) assert.equal(sha(stored), entry.retained_sha256, entry.retained_name);
  const bytes = entry.encoding === 'utf8-json-string' ? Buffer.from(JSON.parse(stored).value, 'utf8') : stored;
  assert.equal(bytes.length, entry.bytes, entry.name);
  assert.equal(sha(bytes), entry.sha256, entry.name);
  rawBytes += bytes.length;
}
assert.equal(raw.length, 72);
assert.equal(rawBytes, 521317);
const oldBindings = json(retained + '/raw/bindings.json');
const exactBindings = Object.entries(oldBindings.scoped_bindings).map(([path, expected]) => bound(path, expected));
assert.equal(exactBindings.length, 87);
const paths = git(['ls-files', '-z', '--', 'tools/gomad3', 'tools/gomad3integration', 'tests', 'Makefile', 'go.mod', 'go.sum']).toString().split('\0').filter(Boolean).sort();
const changedDocs = new Set(['tools/gomad3/ARCHITECTURE.md', 'tools/gomad3/CLI.md', 'tools/gomad3/README.md', 'tools/gomad3/SPEC.md', 'tools/gomad3/TUTORIAL.md']);
const closure = paths.map(path => {
  const previous = git(['show', retainedRef + ':' + path]);
  const current = read(path);
  const equal = previous.equals(current);
  assert(equal || changedDocs.has(path), 'retained executable/generator input changed ' + path);
  return {path, retained_sha256: sha(previous), current_sha256: sha(current), bytes: current.length, equal, reused: equal};
});
const changedSinceRetained = git(['diff', '--name-only', retainedRef, 'HEAD', '--', '.', ':!.flow']).toString().trim().split('\n').filter(Boolean).map(path => ({path, retained_sha256: sha(git(['show', retainedRef + ':' + path])), current_sha256: sha(read(path))}));
const descriptor = json('tools/gomad3/toolchain/version/version.json');
assert.equal(descriptor.patch_allowlist.length, 20);
assert.equal(descriptor.overlay_allowlist.length, 79);
const archivePath = 'tools/gomad3/.toolchain/downloads/' + descriptor.archive.name;
const archive = bound(archivePath, '4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1');
assert.equal(statSync(root + '/' + archivePath).size, 35109201);
const inventories = json(retained + '/raw/source-inventories-pinned.json');
assert.equal(inventories.exit, 0);
const inventoryLog = read(retained + '/raw/source-inventories-pinned.stdout').toString();
assert(inventoryLog.includes('platforms=[darwin/arm64 linux/amd64] draw rows=273 seeded rows=86 clock rows=48 goroutine rows=10'));
assert(inventoryLog.includes('--- PASS: TestGFieldCompactSourceInventories'));
assert(!inventoryLog.includes('--- SKIP:'));
const identity = json(retained + '/raw/independent-identity.stdout');
assert(identity.current_golden_matches_candidate && identity.native_guard_and_whole_byte_assertion_unchanged && identity.differences.length === 7);
const identityBindings = Object.entries(identity.source_bindings).map(([path, entry]) => bound('tools/gomad3/' + path, entry.worktree_sha256.replace('sha256:', '')));
assert.equal(read('tools/gomad3/runner/testdata/diagnostic-identity-choices.json').length, 13271);
const preservationPath = '.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/preservation.json';
const preservation = json(preservationPath);
assert.equal(preservation.files.length, 21);
assert(preservation.files.every(entry => entry.equal));
const runtimePath = 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go';
const runtime = read(runtimePath).toString();
assert(read(runtimePath).equals(git(['show', compactRef + ':' + runtimePath])));
function body(name) {
  const match = runtime.match(new RegExp('func ' + name + '\\([^]*?\\n}', 'm'));
  assert(match, 'missing body ' + name);
  return match[0];
}
const direct = {
  gomadClockTickDraw: ['gomadDiagnosticDraws.clockTick++', 'gomadClockTickState +='],
  gomadTimerRand: ['gomadDiagnosticDraws.timer++', 'gomadTimerRandom +='],
  gomadRuntimeRand: ['gomadDiagnosticDraws.runtimeRand++', 'gomadRuntimeRandom.Next()'],
  gomadRuntimeCheapRand: ['gomadDiagnosticDraws.runtimeCheapRand++', 'gomadRuntimeCheapRandom +='],
  gomadChoiceRunqSeeded: ['gomadDiagnosticDraws.runq++', 'gomadChoiceRandom('],
  gomadChoiceRunnextSeeded: ['gomadDiagnosticDraws.scheduler++', 'gomadChoiceRandom('],
  gomadChoiceShuffleSeeded: ['gomadDiagnosticDraws.scheduler++', 'gomadChoiceRandom('],
  gomadChoiceSelectSeeded: ['gomadDiagnosticDraws.selectPoll++', 'gomadChoiceSelectRandom +='],
};
const guardOrder = Object.entries(direct).map(([name, mutations]) => {
  const source = body(name);
  const guard = source.indexOf('gomadDiagnosticCheckSeededDraw(');
  assert(guard >= 0);
  const checked = mutations.map(mutation => {
    const offset = source.indexOf(mutation);
    assert(offset > guard, name + ' mutates before checking ' + mutation);
    return {mutation, offset};
  });
  return {name, body_sha256: sha(source), guard_offset: guard, mutations: checked};
});
const randomBody = body('gomadChoiceRandom');
assert(randomBody.indexOf('gomadSeededDrawCheck()') < randomBody.indexOf('random.Next()'));
const checkBody = body('gomadDiagnosticCheckSeededDraw');
assert(checkBody.includes('gomadDiagnosticEnabled && mp.gomadHostDraw') && checkBody.includes('exit(2)') && checkBody.includes('mp.gomadHostTimed != 0') && checkBody.includes('exit(125)'));
const faultBody = body('gomadDiagnosticAppend');
assert(faultBody.indexOf('getg().m.gomadHostDraw = true') < faultBody.indexOf('gomadRuntimeCheapRand()'));
assert(runtime.includes('perturbValue[:5] == "host:"') && runtime.includes('const gomadDiagnosticPerturbHostTimedPrefix = "host-timed:"'));
const patch = read('tools/gomad3/toolchain/runtime/go1.27.1.patch').toString();
assert(patch.includes('spinning        bool') && patch.includes('+\tgomadHostDraw   bool\n \tblocked         bool'));
assert.equal((patch.match(/previousHostDrawScope/g) ?? []).length, 2);
assert(!patch.includes('diff --git a/src/runtime/mgc'));
for (const file of ['Makefile', 'tools/gomad3/Makefile', 'tools/gomad3/internal/gomadtool/conformance/runtime_campaign.go']) assert(read(file).toString().includes('GOEXPERIMENT=nogreenteagc'), 'classic collector launcher missing ' + file);
assert(read('tools/gomad3/README.md').toString().includes('enlistWorker') && read('tools/gomad3/README.md').toString().includes('Green Tea'));
assert(read('tools/gomad3/SPEC.md').toString().includes('A remaining host-timed seeded draw that only a prohibited collector file can reroute must be recorded as such.'));
const users = [
  bound('.turbo/plans/gomad3-glossary-update.md', '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188'),
  bound('.turbo/technical-debt.md', 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287'),
];
const result = {head, base_commit: base, retained_ref: retainedRef, compact_ref: compactRef, retained_raw_manifest: {path: retained + '/raw-manifest.json', files: raw.length, bytes: rawBytes, sha256: sha(read(retained + '/raw-manifest.json'))}, exact_bindings: exactBindings, archive, allowlists: [20, 79], input_closure: closure, changed_since_retained: changedSinceRetained, positive_inventory: {receipt: retained + '/raw/source-inventories-pinned.json', receipt_sha256: sha(read(retained + '/raw/source-inventories-pinned.json')), log_sha256: sha(read(retained + '/raw/source-inventories-pinned.stdout')), draw_rows: 273, seeded_rows: 86, clock_rows: 48, goroutine_rows: 10, platforms: ['darwin/arm64', 'linux/amd64'], kind: 'materialized supported source sets; no built/native runtime'}, preservation: {path: preservationPath, sha256: sha(read(preservationPath)), files: 21, full_alpha_chain_sites: 30, exact_input_bindings: exactBindings.length}, identity: {receipt: retained + '/raw/independent-identity.stdout', sha256: sha(read(retained + '/raw/independent-identity.stdout')), pointers: identity.differences.map(entry => entry.path), bindings: identityBindings, bytes: 13271, native_qualification: false}, guard_order: guardOrder, user_files: users, canonical: oldBindings.canonical, collector_blocker: 'runtime/mgcpacer.go:gcControllerState.enlistWorker; prohibited collector edit, retained explicit task acceptance alternative', native_status: 'unverified; fn-149 Darwin and fn-128 Linux/CI remain deferred', product_edits: []};
writeFileSync(out + '/source-binding.json', JSON.stringify(result, null, 2) + '\n');
console.log(JSON.stringify({head, raw_files: raw.length, raw_bytes: rawBytes, exact_bindings: exactBindings.length, input_closure: closure.length, unchanged_inputs: closure.filter(entry => entry.equal).length, changed_documents: closure.filter(entry => !entry.equal).map(entry => entry.path), changed_since_retained: changedSinceRetained.map(entry => entry.path), guard_order: guardOrder.length, positive_inventory: result.positive_inventory, archive, product_edits: []}));
