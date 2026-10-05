import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(fileURLToPath(import.meta.url));
const scratch = path.join(root, 'tools/gomad3/.toolchain/fn-110/gfield-compact.gxrPVUi5');
const prior = path.join(root, '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005');
const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const read = name => fs.readFileSync(name);
const json = name => JSON.parse(read(name));
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const command = (tool, argv) => execFileSync(tool, argv, { cwd: root, maxBuffer: 32 << 20 }).toString().trim();
const base = '1b0bc277589d141aca8b534b03135ab3e57fc050';
assert.equal(command('git', ['rev-parse', 'HEAD']), base);
const freeze = json(path.join(scratch, 'freeze.json'));
const final = json(path.join(scratch, 'final-sources.json'));
const current = Object.fromEntries(command('git', ['ls-files', '-z']).split('\0').filter(name => name && !name.startsWith('.flow/') && fs.existsSync(path.join(root, name)) && fs.statSync(path.join(root, name)).isFile()).sort().map(name => [name, hash(read(path.join(root, name)))]));
assert.deepEqual(current, final);
assert.deepEqual(Object.keys(current), Object.keys(freeze.sources));
assert.deepEqual(freeze.sources, json(path.join(prior, 'final-sources.json')));
const preservation = json(path.join(out, 'preservation.json'));
assert.equal(hash(JSON.stringify(current)), preservation.sources_final_sha256);
assert.deepEqual(Object.keys(current).filter(name => current[name] !== freeze.sources[name]), preservation.changes);
assert.equal(preservation.changes.length, 7);
assert.equal(hash(read(path.join(root, 'tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz'))), freeze.archive_sha256);
assert.equal(hash(read(path.join(root, 'tools/gomad3/toolchain/version/version.json'))), freeze.descriptor_sha256);
for (const [name, digest] of Object.entries(freeze.user_hashes)) assert.equal(hash(read(path.join(root, name))), digest);
for (const context of [1, 3]) assert(read(path.join(scratch, `final-U${context}.patch`)).equals(read(path.join(scratch, `frozen-U${context}.patch`))));
const emit = json(path.join(out, 'identity-audit/candidate-emit.receipt.json')).stdout;
const fixture = read(path.join(root, 'tools/gomad3/runner/testdata/diagnostic-identity-choices.json'));
assert(fixture.equals(Buffer.from(emit)));
assert.notEqual(fixture.at(-1), 10);
assert.equal(hash(fixture), 'b82251eb1c7efa8885bb0cf2992c10fe1a98f11fb10e69f27b4e5ba1c6b773d0');
const generated = preservation.changes.filter(name => name.endsWith('_generated.go'));
const generatedFields = {};
for (const name of generated) {
  const diff = command('git', ['diff', '--unified=0', base, '--', name]);
  const lines = diff.split('\n').filter(line => /^[+-](?![+-])/.test(line));
  assert(lines.every(line => /^[-+]\s+(?:ImplementationSourceSHA256|ProducerImplementationSHA256|GuardImplementationSHA256)\s*=/.test(line)), name);
  assert.equal(lines.length, name.endsWith('wire_generated.go') ? 2 : 4);
  generatedFields[name] = lines.length / 2;
}
const overlay = read(path.join(root, 'tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go')).toString();
const previousOverlay = command('git', ['show', base + ':tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go']);
for (const name of ['gomadSimulationDomain', 'gomadSimulationTransportSyscalls']) {
  const expression = new RegExp('\\b' + name + '\\b', 'g');
  const retainedBaseline = previousOverlay.replace(new RegExp('\\.' + name + '\\b', 'g'), '.renamedPrivateField');
  assert.equal([...overlay.matchAll(expression)].length, [...retainedBaseline.matchAll(expression)].length);
}
assert(/func gomadSimulationDomain\(\)/.test(overlay));
const expected = {
  'baseline-generate-validate': 0, 'baseline-red-alignment': 1, 'baseline-inventories': 0,
  'final-patch-regenerate': 0, 'final-generate-validate': 0, 'final-golden-format': 0,
  'final-materialize': 0, 'final-preservation': 0, 'final-focused-frozen': 0,
  'final-pure-host': 0, 'final-pure-host-observations': 0, 'final-pure-host-report': 0, 'final-pure-host-boundaries': 0,
  'final-diagnostic-controls': 1, 'final-diagnostic-observations': 0,
  'final-scoped-vet': 1, 'final-scoped-vet-host-only': 0,
  'final-lint-code-fast': 0, 'final-generate-validate-frozen': 0,
};
const receipts = {};
for (const [label, exit] of Object.entries(expected)) {
  const receipt = json(path.join(out, label + '.json'));
  assert.equal(receipt.exit, exit, label);
  assert.equal(receipt.signal, null, label);
  assert.equal(receipt.error, null, label);
  assert.equal(hash(read(path.join(out, label + '.stdout'))), receipt.stdout_sha256, label);
  assert.equal(hash(read(path.join(out, label + '.stderr'))), receipt.stderr_sha256, label);
  if (!['final-patch-regenerate', 'final-generate-validate', 'final-golden-format'].includes(label)) {
    assert.deepEqual(receipt.source_changes, [], label);
    assert.equal(receipt.sources_before_sha256, receipt.sources_after_sha256, label);
  }
  if (label.startsWith('final-') && !['final-patch-regenerate', 'final-generate-validate', 'final-golden-format'].includes(label)) assert.equal(receipt.sources_after_sha256, preservation.sources_final_sha256, label);
  for (const tool of ['go', 'gofmt']) assert.equal(receipt.tool_sha256[tool], hash(read(path.join(stock, tool))), label);
  receipts[label] = { exit, elapsed_seconds: receipt.elapsed_seconds, receipt_sha256: hash(read(path.join(out, label + '.json'))), stdout_sha256: receipt.stdout_sha256, stderr_sha256: receipt.stderr_sha256 };
}
const focused = read(path.join(out, 'final-focused-frozen.stdout')).toString();
assert.equal((focused.match(/^--- PASS: /gm) ?? []).length, 21);
assert(!/^--- (?:FAIL|SKIP): /m.test(focused));
assert(focused.includes('otherwise unchanged g fields with avoidable alignment edits=0: []'));
assert(focused.includes('platforms=[darwin/arm64 linux/amd64] draw rows=273 seeded rows=86 clock rows=48 goroutine rows=10'));
const red = read(path.join(out, 'baseline-red-alignment.stdout')).toString();
assert(red.includes('otherwise unchanged g fields with avoidable alignment edits=7:'));
const lint = read(path.join(out, 'final-lint-code-fast.stdout')).toString();
assert(lint.includes('55 host packages'));
assert(lint.includes('0 issues.'));
const lintReceipt = json(path.join(out, 'final-lint-code-fast.json'));
assert(lintReceipt.argv.includes('GOLANGCI_LINT_FIX=false'));
assert(lintReceipt.argv.includes('GOLANGCI_LINT_BASE_REV=1b0bc27758'));
const lintErrors = read(path.join(out, 'final-lint-code-fast.stderr')).toString();
assert(lintErrors.includes('diff: 317/0'));
const tools = Object.fromEntries([
  ['go', path.join(stock, 'go'), ['version']],
  ['golangci-lint', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', ['version']],
  ['errortype', '/tmp/fn109-lint-tools.ZdNe1t50/errortype', ['-V=full']],
].map(([name, file, argv]) => [name, { path: file, sha256: hash(read(file)), version: command(file, argv) }]));
const info = { base_commit: base, commits: [], terminal_handles: [], source_count: Object.keys(current).length, sources_final_sha256: preservation.sources_final_sha256, source_stable: true, exact_product_changes: preservation.changes, prior_baseline_binding: { receipt_directory: path.relative(root, prior), all_5076_sources_equal: true }, archive_sha256: freeze.archive_sha256, descriptor_sha256: freeze.descriptor_sha256, fixture: { bytes: fixture.length, final_lf: false, sha256: hash(fixture), canonical_emit_equal: true }, generated_identity_fields_only: generatedFields, measurements: preservation.measurements, original_u3_comparator_bytes: 32652, remaining_u3_gap_bytes: 642, focused_top_level_passes: 21, inventories: { platforms: ['darwin/arm64', 'linux/amd64'], draw: 273, seeded: 86, clock: 48, goroutine: 10, original_negative_controls_passed: 4 }, host: command('uname', ['-s', '-m']), tools, receipts, limitations: ['R8 remains open at U3=33294 versus original32652', 'native Darwin toolchain/runtime diagnostics/qualification and soak not executed', 'Linux/aarch64 stock-Go checks are developmental only; fn-128 owns Linux qualification', 'full host gate remains inconclusive from prior unsupported-host timeout; unchanged hang was not retried', 'three unchanged diagnostics controls fail for missing instrumented toolchain; native guard skips', 'wrong-surface recursive overlay vet receipt retained as inconclusive and corrected host-only vet passes', 'changed-only lint does not resolve the 317 previously reported full-lint issues', '106 inherited deterministicio failures outside selected pure-host scope remain open', 'conductor owns review, commits and all Flow state'] };
fs.writeFileSync(path.join(out, 'closure.json'), JSON.stringify(info, null, 2) + '\n');
console.log('final source stable; exact seven admitted product paths; receipts and tool identities verified; U3 gap=642 bytes');
