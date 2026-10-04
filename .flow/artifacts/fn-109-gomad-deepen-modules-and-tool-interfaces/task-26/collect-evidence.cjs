const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const dir = __dirname;
const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = p => fs.readFileSync(path.join(dir, p), 'utf8');
const old = JSON.parse(read('old-source.json'));
const delta = JSON.parse(read('verified-source.json'));
const current = { ...old, ...delta.changed };
function findings(name) {
  const lines = read(name).split('\n');
  const result = [];
  for (let i = 0; i < lines.length; i++) {
    const m = lines[i].match(/^(tools\/gomad3\/[^:]+):(\d+):(\d+): (.*) \((\w+)\)$/);
    if (!m) continue;
    result.push({ path: m[1], line: +m[2], column: +m[3], message: m[4], linter: m[5], source: lines[i + 1]?.trim() });
  }
  return result;
}
const before = findings('baseline-lint.log');
const after = findings('final-lint.log');
const key = f => JSON.stringify([f.path, f.message, f.linter, f.source]);
const remaining = before.slice();
const inherited = [], introduced = [];
for (const finding of after) {
  const index = remaining.findIndex(f => key(f) === key(finding));
  if (index < 0) introduced.push(finding);
  else inherited.push({ before: remaining.splice(index, 1)[0], after: finding });
}
const lint = { before_count: before.length, after_count: after.length, inherited, introduced, resolved: remaining, matching: 'multiset of path, complete diagnostic, linter and exact trimmed source excerpt; before/after locations retained', owners: [...new Set(after.map(f => f.path))].sort() };
fs.writeFileSync(path.join(dir, 'lint-delta.json'), JSON.stringify(lint, null, 2) + '\n');
const receipts = fs.readdirSync(dir).filter(n => n.endsWith('.receipt.json')).sort().map(name => ({ path: path.join(dir, name), sha256: hash(fs.readFileSync(path.join(dir, name))), ...JSON.parse(read(name)) }));
const provenancePaths = ['tools/gomad3/cmd/gomad/e2e_test.go', 'tools/gomad3/cmd/gomad/internal/cli/doctor_test.go', 'tools/gomad3/cmd/gomad/internal/cli/cli_test.go', 'tools/gomad3/runner/campaign_options.go', 'tools/gomad3/runner/portable_plan_test.go', 'tools/gomad3/runner/preparation_owner_test.go'];
function functionBytes(bytes, name) {
  const start = bytes.indexOf('func ' + name + '(');
  if (start < 0) throw new Error('missing function ' + name);
  const next = bytes.indexOf('\nfunc ', start + 1);
  return bytes.slice(start, next < 0 ? bytes.length : next);
}
const git = require('child_process');
const unchangedBodies = {};
for (const name of ['TestRunAnalyzeClassifiesRealReadonlyModuleFailureAsInvalidInput', 'TestRunDoctorReportsAvailableContractAsJSON']) {
  const p = 'tools/gomad3/cmd/gomad/internal/cli/cli_test.go';
  const baseline = git.execFileSync('git', ['show', '984fa118347ebc7b39b7080dd5b9e95e941a00d4:' + p], { cwd: root }).toString();
  const now = fs.readFileSync(path.join(root, p), 'utf8');
  unchangedBodies[name] = { source: p, baseline_sha256: hash(functionBytes(baseline, name)), final_sha256: hash(functionBytes(now, name)) };
}
const receiptIndex = receipts.map(r => ({ path: r.path, sha256: r.sha256, exit_code: r.exit_code, elapsed_seconds: r.elapsed_seconds, source_freeze: path.basename(r.source_freeze), source_freeze_sha256: r.source_freeze_sha256, stability_before: r.stability_before, stability_after: r.stability_after }));
const evidence = { task: 'fn-109-gomad-deepen-modules-and-tool-interfaces.26', status: 'SOURCE_PROGRESS_ONLY', base_commit: '984fa118347ebc7b39b7080dd5b9e95e941a00d4', commits: [], tests: receipts.map(r => r.command), prs: [], receipts: receiptIndex, lint_delta: path.join(dir, 'lint-delta.json'), source: { base_freeze: path.join(dir, 'old-source.json'), final_delta: path.join(dir, 'verified-source.json'), changed_paths: Object.keys(delta.changed), production_cli_before_sha256: old['tools/gomad3/cmd/gomad/internal/cli/cli.go'], production_cli_final_sha256: current['tools/gomad3/cmd/gomad/internal/cli/cli.go'], unchanged_inputs: Object.fromEntries(provenancePaths.map(p => [p, { baseline_sha256: old[p], final_sha256: current[p] }])), unchanged_failing_test_bodies: unchangedBodies }, open_gates: ['ordinary CLI complete package and end-to-end gate (absent patched toolchain / unsupported linux/arm64)', 'expanded portable-plan native execution tests', 'darwin-only fixed diagnostic identity goldens (skipped by existing platform guard)', '54 inherited unfiltered CLI golangci findings', 'original task5 and task4/predecessor acceptance', 'R6/R18/R19 and task21 full/native qualification on darwin/arm64 and linux/amd64', 'conductor independent source review and progress commit', 'formal full-green-tree review'], review: 'deferred to conductor; no verdict', delegates: 0, live_handles: [] };
fs.writeFileSync(path.join(dir, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n');
console.log(JSON.stringify({ before: before.length, after: after.length, introduced: introduced.length, resolved: remaining.length, receipts: receipts.length, all_source_stable: receipts.every(r => r.stability_before && r.stability_after), unchanged_failing_test_bodies: unchangedBodies }));
if (introduced.length || remaining.length || receipts.some(r => !r.stability_before || !r.stability_after || r.error)) process.exitCode = 1;
