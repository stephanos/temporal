import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const out = path.dirname(fileURLToPath(import.meta.url));
const root = execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = name => fs.readFileSync(path.join(root, name));
const baseline = JSON.parse(fs.readFileSync(path.join(out, 'freeze.json')));
assert.equal(execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(), baseline.base_commit);
const allowed = [
  'tools/gomad3/toolchain/runtime/go1.27.1.patch',
  'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go',
  'tools/gomad3/choice/internal/wire/wire_generated.go',
  'tools/gomad3/target/internal/livecap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go',
  'tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go',
  'tools/gomad3/runner/testdata/diagnostic-identity-choices.json',
].sort();
const before = Object.fromEntries(Object.keys(baseline.sources).sort().map(name => [name, hash(read(name))]));
const finalSources = JSON.parse(fs.readFileSync(path.join(out, 'final-sources.json')));
assert.deepEqual(before, finalSources, 'final frozen source bindings differ from worktree');
const changed = Object.keys(before).filter(name => before[name] !== baseline.sources[name]).sort();
assert.deepEqual(changed, allowed);
const productDiff = execFileSync('git', ['diff', '--name-only', baseline.base_commit], { encoding: 'utf8' })
  .trim().split('\n').filter(name => name && !name.startsWith('.flow/')).sort();
assert.deepEqual(productDiff, allowed);
for (const [name, expected] of Object.entries(baseline.tools)) assert.equal(hash(fs.readFileSync(name)), expected, name);
for (const [name, expected] of Object.entries(baseline.user_files)) assert.equal(hash(read(name)), expected, name);
const descriptor = 'tools/gomad3/toolchain/version/version.json';
assert(read(descriptor).equals(execFileSync('git', ['show', `${baseline.base_commit}:${descriptor}`])), 'descriptor changed');
const policy = JSON.parse(read(descriptor));
assert.equal(policy.patch_allowlist.length, 20);
assert.equal(policy.overlay_allowlist.length, 79);
const emitted = fs.readFileSync(path.join(out, 'final-identity-emit.stdout'));
const golden = read('tools/gomad3/runner/testdata/diagnostic-identity-choices.json');
assert(golden.equals(emitted), 'golden differs from independently emitted bytes');
assert.notEqual(golden.at(-1), 10, 'golden has a final LF');
const envKeys = new Set(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS',
  'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'PATH', 'GOEXPERIMENT', 'GOROOT', 'GOMADSEED',
  'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOCACHE', 'GOMODCACHE']);
const receipts = [];
let environmentObjects = 0;
function inspectEnvironment(value) {
  if (!value || typeof value !== 'object') return;
  for (const [key, child] of Object.entries(value)) {
    if (key === 'environment' && child && !Array.isArray(child) && typeof child === 'object') {
      environmentObjects++;
      assert(Object.keys(child).every(name => envKeys.has(name)), 'unexpected recorded environment key');
    } else inspectEnvironment(child);
  }
}
function inspectDirectory(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const name = path.join(directory, entry.name);
    if (entry.isDirectory()) { inspectDirectory(name); continue; }
    if (!entry.name.endsWith('.json')) continue;
    const value = JSON.parse(fs.readFileSync(name));
    inspectEnvironment(value);
    if (!Array.isArray(value.argv)) continue;
    const label = name.slice(0, -5);
    const stdout = label + '.stdout';
    const stderr = label + '.stderr';
    if (!fs.existsSync(stdout) || !fs.existsSync(stderr)) continue;
    assert.equal(hash(fs.readFileSync(stdout)), value.stdout_sha256, stdout);
    assert.equal(hash(fs.readFileSync(stderr)), value.stderr_sha256, stderr);
    const exit = value.exit ?? value.exit_code;
    assert(Number.isInteger(exit), `nonterminal receipt: ${entry.name}`);
    receipts.push({ name: path.relative(out, name), exit,
      stable: value.sources_before_sha256 === value.sources_after_sha256,
      source_changes: value.source_changes ?? [],
      stdout_bytes: fs.statSync(stdout).size, stderr_bytes: fs.statSync(stderr).size });
  }
}
inspectDirectory(out);
function goResults(label) {
  const tests = new Map();
  const packages = new Map();
  for (const line of fs.readFileSync(path.join(out, `${label}.stdout`), 'utf8').split('\n')) {
    let event;
    try { event = JSON.parse(line); } catch { continue; }
    if (!event.Action || !event.Package) continue;
    if (event.Test) {
      const key = `${event.Package}:${event.Test}`;
      const entry = tests.get(key) ?? { test: event.Test, output: '', errors: [] };
      if (['pass', 'fail', 'skip'].includes(event.Action)) entry.result = event.Action;
      if (event.Action === 'output') entry.output += event.Output ?? '';
      if (event.OutputType === 'error') entry.errors.push(event.Output);
      tests.set(key, entry);
    } else if (['pass', 'fail', 'skip'].includes(event.Action)) packages.set(event.Package, event.Action);
  }
  const counts = {};
  const topLevelCounts = {};
  for (const result of ['pass', 'fail', 'skip']) {
    counts[result] = [...tests.values()].filter(entry => entry.result === result).length;
    topLevelCounts[result] = [...tests.values()].filter(entry => entry.result === result && !entry.test.includes('/')).length;
  }
  return { tests, summary: { label, counts, top_level_counts: topLevelCounts,
    package_results: Object.fromEntries(packages),
    unfinished: [...tests].filter(([, entry]) => !entry.result).map(([key]) => key) } };
}
const hostBaseline = goResults('baseline-host-developmental');
const finalPure = goResults('final-pure-host');
const finalDiagnostic = goResults('final-diagnostic-controls');
const unmatchedFailures = [];
const changedVerdicts = [];
const changedErrors = [];
for (const final of [finalPure, finalDiagnostic]) {
  for (const [key, entry] of final.tests) {
    const baseEntry = hostBaseline.tests.get(key);
    if (entry.result && baseEntry?.result !== entry.result) {
      changedVerdicts.push({ label: final.summary.label, test: key, final: entry.result, baseline: baseEntry?.result ?? null });
    }
    if (entry.result === 'fail' && hostBaseline.tests.get(key)?.result !== 'fail') {
      unmatchedFailures.push({ label: final.summary.label, test: key, output: entry.output });
    }
    if (entry.result === 'fail' && JSON.stringify(entry.errors) !== JSON.stringify(baseEntry?.errors)) {
      changedErrors.push({ label: final.summary.label, test: key, baseline: baseEntry?.errors, final: entry.errors });
    }
  }
}
const patches = {};
for (const phase of ['baseline', 'final']) {
  for (const context of [1, 3]) {
    const name = `${phase}-U${context}.patch`;
    const bytes = fs.readFileSync(path.join(out, name));
    patches[name] = { bytes: bytes.length, lines: bytes.toString().split('\n').length - 1, sha256: hash(bytes) };
  }
}
assert(read('tools/gomad3/toolchain/runtime/go1.27.1.patch').equals(fs.readFileSync(path.join(out, 'final-U1.patch'))));
assert.equal(patches['baseline-U1.patch'].bytes, 29015);
assert.equal(patches['baseline-U3.patch'].bytes, 38362);
const after = Object.fromEntries(Object.keys(before).map(name => [name, hash(read(name))]));
assert.deepEqual(after, before, 'product source changed during root verification');
console.log(JSON.stringify({ base_commit: baseline.base_commit, product_source_count: Object.keys(before).length,
  product_changes: changed, product_stable_during_audit: true, user_files_preserved: true,
  tools_preserved: true, allowlists: { patch: 20, overlay: 79 }, environment_objects: environmentObjects,
  unexpected_environment_keys: 0, golden_bytes: golden.length, golden_matches_independent_emit: true,
  patches, original_R8_U3_baseline: 32652, R8_remaining_bytes: patches['final-U3.patch'].bytes - 32652,
  go_observations: [hostBaseline.summary, finalPure.summary, finalDiagnostic.summary],
  failures_without_matching_baseline_failure: unmatchedFailures,
  final_verdicts_different_from_baseline: changedVerdicts, final_raw_errors_different_from_baseline: changedErrors,
  receipts }, null, 2));
