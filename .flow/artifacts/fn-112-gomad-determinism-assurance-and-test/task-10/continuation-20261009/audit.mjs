import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const packet = '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10';
const source = `${packet}/source-acceptance-20261009`;
const diagnostic = `${packet}/diagnostic-writes-20261009`;
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const read = file => fs.readFileSync(path.resolve(root, file));
const json = file => JSON.parse(read(file));
const git = (...args) => execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim();
const errors = [];
const check = (condition, detail) => { if (!condition) errors.push(detail); };
const evidence = json(`${diagnostic}/evidence.json`);
const prior = json(`${source}/evidence.json`);
const documents = json(`${source}/sealed-source-audit.json`);
const paths = git('ls-files', '-z', 'tools/gomad3', 'tools/gomad3integration',
  '.github/workflows/gomad3.yml', '.github/.golangci.yml', 'Makefile',
  'AGENTS.md', 'MILESTONES.md', 'cmd/tools/lintcode').split('\0').filter(Boolean).sort();
const bindings = Object.fromEntries(paths.map(file => [file, sha(read(file))]));
const fingerprintOf = values => sha('{' + Object.entries(values).map(([file, digest]) =>
  JSON.stringify(file) + ': ' + JSON.stringify(digest)).join(', ') + '}');
const currentFingerprint = fingerprintOf(bindings);
const milestonesDiff = git('diff', '--', 'MILESTONES.md');
const sourceBindings = { ...bindings, 'MILESTONES.md': sha(execFileSync('git',
  ['show', 'HEAD:MILESTONES.md'], { cwd: root })) };
const fingerprint = fingerprintOf(sourceBindings);
check(!milestonesDiff || milestonesDiff.includes('fn-109.52'), 'unexpected current milestone edit');
const reused = [];
for (const name of ['final-command-soak-set', 'affected-vet', 'standalone-errortype',
  'architecture-source-sets', 'final-validate', 'format-check', 'final-configured-lint',
  'make-fast-admission', 'make-gomad-original-base']) {
  const gate = evidence.gates[name];
  check(gate.source_before_sha256 === fingerprint && gate.source_after_sha256 === fingerprint,
    `${name}: current source fingerprint mismatch`);
  check(gate.terminal && gate.source_unchanged, `${name}: nonterminal or mixed source`);
  for (const stream of ['stdout', 'stderr']) {
    check(sha(read(`${diagnostic}/${name}.${stream}`)) === gate[`${stream}_sha256`],
      `${name}: ${stream} hash mismatch`);
  }
  for (const [file, expected] of Object.entries(gate.tools)) {
    check(sha(read(file)) === expected, `${name}: tool hash mismatch ${file}`);
  }
  reused.push({ packet: `${diagnostic}/${name}.json`, command: gate.command,
    exit_code: gate.exit_code, elapsed_seconds: gate.elapsed_seconds,
    source_fingerprint: gate.source_after_sha256,
    stdout_sha256: gate.stdout_sha256, stderr_sha256: gate.stderr_sha256 });
}
const docBindings = [];
for (const [file, expected] of Object.entries(documents.inputs_sha256)) {
  const observed = sha(read(file));
  check(observed === expected, `retained guide binding mismatch ${file}`);
  docBindings.push({ file, expected_sha256: expected, current_sha256: observed });
}
check(documents.errors.length === 0, 'retained source guide audit errors');
for (const [file, data] of Object.entries(evidence.source_changes)) {
  check(sha(read(file)) === data.after_sha256, `preserved diagnostic source mismatch ${file}`);
}
for (const [file, data] of Object.entries(evidence.protected_user_files)) {
  check(sha(read(file)) === data.current_sha256_observed_by_worker, `protected file mismatch ${file}`);
}
const scopedRaw = read(`${diagnostic}/final-configured-lint.stdout`).toString();
const originalRaw = read(`${diagnostic}/make-gomad-original-base.stdout`).toString();
const parse = value => Array.from(value.matchAll(/^(.+\.go):(\d+):(\d+): (.+) \(([^)]+)\)$/gm), match => {
  const [, file, line, column, message, linter] = match;
  const lines = read(file).toString().split('\n');
  return { file, line: Number(line), column: Number(column), message, linter,
    statement: lines[Number(line) - 1].trim(), file_sha256: sha(read(file)) };
});
const residual = parse(scopedRaw);
const original = parse(originalRaw);
check(residual.length === 59 && residual.every(item => item.linter === 'errcheck'), 'scoped residual inventory changed');
check(original.length === 204, 'original-base residual inventory changed');
const byFile = items => items.reduce((result, item) => {
  result[item.file] = (result[item.file] || 0) + 1;
  return result;
}, {});
const predecessors = [4, 5, 6, 7, 15, 16].map(number => {
  const id = `fn-112-gomad-determinism-assurance-and-test.${number}`;
  const data = JSON.parse(execFileSync('/home/agent/.codex/scripts/flowctl', ['show', id, '--json'],
    { cwd: root, encoding: 'utf8' }));
  check(data.status === 'done', `predecessor ${id} is ${data.status}`);
  return { id, status: data.status, task_sha256: sha(read(data.spec_path)) };
});
const references = [
  `${source}/handover.md`, `${source}/evidence.json`, `${source}/sealed-source-audit.json`,
  `${source}/final-guides/guide-audit.json`, `${diagnostic}/handover.md`, `${diagnostic}/evidence.json`,
  '.flow/artifacts/native-scope-transfer-2026-10-07.md',
  '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.46.md',
  '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.50.md',
  '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.51.md',
];
const output = {
  task: 'fn-112-gomad-determinism-assurance-and-test.10', status: 'in_progress',
  head: git('rev-parse', 'HEAD'), branch: git('branch', '--show-current'),
  audited_at: new Date().toISOString(), acceptance_verdict: null, review_verdict: null,
  source_fingerprint: fingerprint, current_worktree_fingerprint: currentFingerprint,
  root_lifecycle_milestones_diff: milestonesDiff, source_path_count: paths.length,
  errors, reused_gate_receipts: reused, guide_input_bindings: docBindings,
  residual59: residual, residual59_by_file: byFile(residual),
  original_base: evidence.lint.original_base, original_base_exit: 2,
  original_base_findings: original.length, original_base_by_file: byFile(original),
  original_base_integrated_errortype_reached: false,
  corrected_source_tests: evidence.test_counts['final-command-soak-set'],
  predecessor_statuses: predecessors,
  current_guides: { errors: documents.errors, help_entries: documents.command_help_inventory.length,
    local_links: documents.links.length, all_links_resolve: documents.links.every(item => item.resolves),
    legacy_checker_exit: prior.gates['final-guide-audit'].exit_code,
    legacy_checker_diagnostics: json(`${source}/final-guides/guide-audit.json`).errors.length },
  references: references.map(file => ({ file, sha256: sha(read(file)) })),
  source_changes: [], commits: [], tests: [], prs: [], live_command_handles: [],
  native_qualification_claim: false, native_bound: null,
  blockers: [
    { owner: 'fn-109-gomad-deepen-modules-and-tool-interfaces.52',
      gate: 'affected configured gomadtool lint', count: 59, status: 'red' },
    { owner: 'fn-109 correction/source owners, reconciled by fn-109.21',
      gate: 'configured original-base lint', count: 204, status: 'red' },
    { owner: 'root for fn-112.10', gate: 'formal independent source implementation review', status: 'pending green source gates' },
  ],
  audit_script_sha256: sha(read(`${packet}/continuation-20261009/audit.mjs`)),
};
console.log(JSON.stringify(output, null, 2));
process.exitCode = errors.length ? 1 : 0;
