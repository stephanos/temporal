import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

// Task-local independent calculation. It never writes a file or executes Go.
const root = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const base = '1147416b2e';
const args = process.argv.slice(2);
const emit = args.includes('--emit');
const sourceRef = args.find(arg => arg.startsWith('--source-ref='))?.slice(13) ?? 'worktree';
const expectedSource = args.find(arg => arg.startsWith('--expected-source='))?.slice(18);
assert(args.every(arg => arg === '--emit' || arg.startsWith('--source-ref=') || arg.startsWith('--expected-source=')), 'unknown argument');
assert(sourceRef === 'worktree' || sourceRef === base, 'source ref must be worktree or the fixed baseline');
const prefix = 'tools/gomad3/';
const read = (relative, ref = sourceRef) => ref === 'worktree'
  ? fs.readFileSync(path.join(root, prefix, relative))
  : execFileSync('git', ['show', `${ref}:${prefix}${relative}`], { cwd: root, maxBuffer: 16 << 20 });
const hash = (...parts) => crypto.createHash('sha256').update(Buffer.concat(parts.map(value => Buffer.isBuffer(value) ? value : Buffer.from(value)))).digest();
const digest = value => 'sha256:' + hash(value).toString('hex');
const raw = value => Buffer.from(value.replace(/^sha256:/, ''), 'hex');
const u64 = value => { const bytes = Buffer.alloc(8); bytes.writeBigUInt64BE(BigInt(value)); return bytes; };
const pick = (object, names) => Object.fromEntries(names.split(' ').filter(name => name in object).map(name => [name, object[name]]));
function sorted(value) {
  if (value === null || typeof value !== 'object') return value;
  if (Array.isArray(value)) return value.map(sorted);
  return Object.fromEntries(Object.keys(value).sort().map(key => [key, sorted(value[key])]));
}
const canonical = value => JSON.stringify(sorted(value));
const sourcePaths = [
  'choice/schema/choicewire.json', 'choice/schema/choicewire.go.tmpl',
  'choice/schema/choicewire_runtime.go.tmpl', 'toolchain/runtime/overlay/src/runtime/gomad.go',
  'toolchain/runtime/go1.27.1.patch', 'choice/trace.go', 'choice/tape.go',
];
const contractPaths = [
  'runner/runner_test.go', 'runner/diagnostic_identity_test.go',
  'record/identity.go', 'record/record.go', 'internal/canonicaljson/canonical.go',
  'runner/portable_plan.go', 'choice/trace.go', 'choice/tape.go',
  'choice/schema/choicewire.json', 'choice/schema/choicewire.go.tmpl',
  'internal/gomadtool/generation/protocol/protocol.go',
];
const goldenPath = 'runner/testdata/diagnostic-identity-choices.json';
const plainPath = 'runner/testdata/diagnostic-identity-plain.json';
const generatedPath = 'choice/internal/wire/wire_generated.go';
const bindings = {};
for (const relative of new Set([...sourcePaths, ...contractPaths, goldenPath, plainPath, generatedPath])) {
  bindings[relative] = { base_sha256: digest(read(relative, base)), selected_sha256: digest(read(relative)), worktree_sha256: digest(read(relative, 'worktree')) };
}
// A changed owner contract needs a new reviewed calculator, not an inferred refresh.
for (const relative of contractPaths) assert(read(relative, 'worktree').equals(read(relative, base)), `owner contract changed: ${relative}`);
assert(read(plainPath, 'worktree').equals(read(plainPath, base)), 'plain fixture changed');
const original = read(goldenPath, base).toString('utf8');
assert(!/[^\x00-\x7f]/.test(original), 'calculator JSON subset requires ASCII fixture strings');
const document = JSON.parse(original);
const manifest = document.artifact;
const checks = {};
assert.equal(canonical(document), original);
checks.old_fixture_canonical_bytes = true;

// Source assertions retain the fake producer and actual guarded assertion.
const runnerSource = read('runner/runner_test.go', 'worktree').toString('utf8');
const preparer = runnerSource.slice(runnerSource.indexOf('func newFakePreparer('), runnerSource.indexOf('func profileFakePreparer('));
const fakeKey = preparer.match(/BuildKey: "([a-f0-9]{64})"/)?.[1];
assert.equal(fakeKey, manifest.toolchain.build_key, 'fixture fake BuildKey disagrees with current newFakePreparer');
assert(preparer.includes('[]byte("fake prepared target")'));
assert.equal(digest('fake prepared target'), manifest.target.sha256);
const diagnosticSource = read('runner/diagnostic_identity_test.go', 'worktree').toString('utf8');
assert(diagnosticSource.includes('runtime.GOOS != "darwin" || runtime.GOARCH != "arm64"'));
assert(diagnosticSource.includes('!bytes.Equal(expected, encoded)'));
assert(diagnosticSource.includes('&explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: 1 << 20, exitCode: 1}'));
const executor = runnerSource.slice(runnerSource.indexOf('func (executor *explorationExecutor) Run('), runnerSource.indexOf('// forceTestChoiceRank'));
assert(executor.includes('Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: max(executor.alternatives, 2)'));
assert.equal(manifest.toolchain.target_goos, 'darwin');
assert.equal(manifest.toolchain.target_goarch, 'arm64');

const schema = JSON.parse(read('choice/schema/choicewire.json'));
assert.equal(schema.version, 3);
assert.equal(schema.trace.record_bytes, 96);
assert.equal(schema.tape.header_bytes, 264);
assert.equal(schema.tape.record_bytes, 96);
assert.equal(schema.tape.checksum_offset, 232);
const alternatives = [hash('choice/0/alternative/0'), hash('choice/0/alternative/1')];
const ordered = [...alternatives].sort(Buffer.compare);
const record = Buffer.alloc(schema.trace.record_bytes);
record[8] = schema.kinds.runnable;
record[9] = schema.flags.decision;
record.writeUInt32BE(2, 12);
record.writeUInt32BE(ordered.findIndex(identity => identity.equals(alternatives[0])), 16);
alternatives[0].copy(record, 32);
hash('gomad3-choice-alternative-set/v1', '\0', u64(2), ...ordered).copy(record, 64);
assert.equal(digest(record), manifest.choice_profile.trace.sha256);
checks.old_trace_sha256 = true;

function tapeDigest(implementation) {
  const header = Buffer.alloc(schema.tape.header_bytes);
  Buffer.from(schema.tape.magic).copy(header);
  header.writeUInt32BE(schema.version, 8);
  header.writeUInt32BE(header.length, 12);
  header.writeUInt32BE(record.length, 16);
  header.writeBigUInt64BE(BigInt(header.length + record.length), 24);
  header.writeBigUInt64BE(1n, 32);
  raw(manifest.choice_profile.trace.sha256).copy(header, 40);
  raw(manifest.target.sha256).copy(header, 72);
  raw(implementation).copy(header, 104);
  raw(fakeKey).copy(header, 136);
  hash('gomad3-choice-platform/v1', '\0', manifest.toolchain.target_goos, '\0', manifest.toolchain.target_goarch).copy(header, 168);
  hash(record).copy(header, 200);
  hash(header.subarray(0, schema.tape.checksum_offset)).copy(header, schema.tape.checksum_offset);
  return digest(Buffer.concat([header, record]));
}

function projectionDigests(value) {
  assert(!value.simulation_profile && !value.minimization && !value.target.capability_manifest && !value.io_profile.read_only_mounts);
  const target = pick(value.target, 'kind sha256 size argv build_tags adapters compatibility build_info capability_mode');
  const io = pick(value.io_profile, 'name implementation_sha256 inventory inventory_sha256');
  if (value.io_profile.transcript) io.transcript = pick(value.io_profile.transcript, 'schema sha256 bytes records');
  const choice = pick(value.choice_profile, 'name implementation_sha256');
  choice.trace = pick(value.choice_profile.trace, 'schema sha256 bytes records branching_records terminal_state limit tape_sha256 decisions');
  const world = {
    initial: pick(value.world.initial, 'schema raw_sha256 semantic_digest'),
    final: pick(value.world.final, 'schema raw_sha256 semantic_digest'),
    transitions: pick(value.world.transitions, 'schema raw_sha256 count transcript_digest'),
    adapters: value.world.adapters.length ? value.world.adapters : null,
    terminal: value.world.terminal,
  };
  const outcome = pick(value.outcome, 'domain reason termination exit_code signal deadline');
  const common = { schema_version: value.schema_version, toolchain: value.toolchain, target, io_profile: io, choice_profile: choice, world, outcome };
  const failure = { ...common, environment: value.environment.filter(entry => entry.name !== 'GOMADSEED'), stdout_sha256: value.streams.stdout.full_sha256, stderr_sha256: value.streams.stderr.full_sha256 };
  const streams = Object.fromEntries(['stdout', 'stderr'].map(name => [name, pick(value.streams[name], 'retained_sha256 full_sha256 total_bytes retained_bytes discarded_bytes truncated')]));
  const execution = { ...common, runner: value.runner, environment: value.environment, limits: value.limits, seed: value.seed, streams };
  return {
    failure: 'sha256:' + hash('gomad3-failure-signature-v1', '\0', canonical(failure)).toString('hex'),
    record: 'sha256:' + hash('gomad3-execution-record-v1', '\0', canonical(execution)).toString('hex'),
  };
}

assert.equal(tapeDigest(manifest.choice_profile.implementation_sha256), manifest.choice_profile.trace.tape_sha256);
checks.old_tape_sha256 = true;
const retainedProjection = projectionDigests(manifest);
assert.equal(retainedProjection.failure, manifest.outcome.failure_signature);
checks.old_failure_signature = true;
assert.equal(retainedProjection.record, manifest.record_hash);
checks.old_record_hash = true;
assert.equal(digest(canonical(document.portable_plan)), document.portable_plan_sha256);
checks.old_portable_plan_sha256 = true;

function sourceFingerprint(ref) {
  const inputs = sourcePaths.map(relative => read(relative, ref));
  return hash('gomad3-choice-implementation-source-v2', ...inputs.flatMap(bytes => [u64(bytes.length), bytes]));
}
function generatedFingerprint(ref) {
  const source = read(generatedPath, ref).toString('utf8');
  const literals = source.match(/ImplementationSourceSHA256 = \[DigestBytes\]byte\{([^\n]+)\}/)?.[1];
  assert(literals, 'generated choice identity is absent');
  const values = [...literals.matchAll(/'((?:\\.|[^'])+)'/gu)].map(match => {
    const value = match[1];
    if (/^\\x[0-9a-fA-F]{2}$/.test(value) || /^\\u[0-9a-fA-F]{4}$/.test(value) || /^\\U[0-9a-fA-F]{8}$/.test(value)) return parseInt(value.slice(2), 16);
    if (/^\\[0-7]{3}$/.test(value)) return parseInt(value.slice(1), 8);
    const escapes = { '\\a': 7, '\\b': 8, '\\t': 9, '\\n': 10, '\\v': 11, '\\f': 12, '\\r': 13, '\\\\': 92, "\\'": 39, '\\"': 34 };
    if (value in escapes) return escapes[value];
    assert.equal([...value].length, 1, `unsupported rune ${value}`);
    return value.codePointAt(0);
  });
  assert.equal(values.length, 32);
  assert(values.every(value => value >= 0 && value <= 255));
  return Buffer.from(values);
}
const implementationFor = fingerprint => 'sha256:' + hash('gomad3-choice-implementation-v2', '\0', fingerprint, raw(fakeKey)).toString('hex');
const baselineFingerprint = sourceFingerprint(base);
assert(baselineFingerprint.equals(generatedFingerprint(base)), 'base generated source fingerprint is stale');
const selectedFingerprint = sourceFingerprint(sourceRef);
assert(selectedFingerprint.equals(generatedFingerprint(sourceRef)), 'selected generated source fingerprint is stale; wait for generation freeze');
if (expectedSource) assert.equal(selectedFingerprint.toString('hex'), expectedSource, 'selected source differs from supplied freeze');
const retainedImplementation = manifest.choice_profile.implementation_sha256;
const selectedImplementation = implementationFor(selectedFingerprint);
manifest.choice_profile.implementation_sha256 = selectedImplementation;
document.campaign_plan.choice_profile.implementation_sha256 = selectedImplementation;
document.portable_plan.campaign.choice_profile.implementation_sha256 = selectedImplementation;
manifest.choice_profile.trace.tape_sha256 = tapeDigest(selectedImplementation);
const updated = projectionDigests(manifest);
manifest.outcome.failure_signature = updated.failure;
manifest.record_hash = updated.record;
document.portable_plan_sha256 = digest(canonical(document.portable_plan));
const differences = [];
function diff(left, right, pointer = '') {
  if (left === null || right === null || typeof left !== 'object' || typeof right !== 'object') {
    if (left !== right) differences.push({ path: pointer, old: left, new: right });
    return;
  }
  assert.deepEqual(Object.keys(left).sort(), Object.keys(right).sort(), `changed object shape at ${pointer}`);
  for (const key of Object.keys(left)) diff(left[key], right[key], `${pointer}/${key}`);
}
diff(JSON.parse(original), document);
const allowed = [
  '/artifact/choice_profile/implementation_sha256', '/artifact/choice_profile/trace/tape_sha256',
  '/artifact/outcome/failure_signature', '/artifact/record_hash',
  '/campaign_plan/choice_profile/implementation_sha256',
  '/portable_plan/campaign/choice_profile/implementation_sha256', '/portable_plan_sha256',
];
assert.deepEqual(differences.map(change => change.path).sort(), [...allowed].sort(), 'delta must be exactly seven identity-derived fields');
const candidate = canonical(document);
const currentGolden = read(goldenPath, 'worktree').toString('utf8');
assert(currentGolden === original || currentGolden === candidate, 'current product golden differs from baseline and derived candidate');
for (const [relative, binding] of Object.entries(bindings)) assert.equal(digest(read(relative, 'worktree')), binding.worktree_sha256, `worktree changed during calculation: ${relative}`);
const report = {
  evidence: 'independent source/schema calculation only; no native Darwin test execution',
  base_commit: execFileSync('git', ['rev-parse', base], { cwd: root, encoding: 'utf8' }).trim(),
  source_ref: sourceRef, self_checks: checks,
  plain_fixture_unchanged: true, fake_build_key: fakeKey,
  owner_contracts_unchanged: contractPaths, native_guard_and_whole_byte_assertion_unchanged: true,
  base_source_fingerprint: baselineFingerprint.toString('hex'),
  base_generated_source_fingerprint: generatedFingerprint(base).toString('hex'),
  base_expected_implementation: implementationFor(baselineFingerprint),
  retained_golden_implementation: retainedImplementation,
  golden_was_stale_at_base: implementationFor(baselineFingerprint) !== retainedImplementation,
  selected_source_fingerprint: selectedFingerprint.toString('hex'), selected_implementation: selectedImplementation,
  candidate_fixture_sha256: digest(candidate), current_golden_matches_candidate: currentGolden === candidate,
  differences, source_bindings: bindings,
  helper_sha256: digest(fs.readFileSync(fileURLToPath(import.meta.url))),
  product_writes: 0,
};
process.stdout.write(emit ? candidate : JSON.stringify(report, null, 2) + '\n');
