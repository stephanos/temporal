import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {repo, out, base, original, hash} from './run.mjs';

const read = name => fs.readFileSync(path.join(out, name), 'utf8');
const receipts = fs.readdirSync(out).filter(name => name.endsWith('-receipt.json')).map(name => {
  const data = JSON.parse(read(name));
  if (!Number.isInteger(data.child_exit_code) || data.signal || data.error || !data.source_unchanged || !data.tools_unchanged || !data.controls_unchanged) throw Error('nonterminal or mutable binding: ' + name);
  for (const binding of [data.source, data.tools, data.controls]) if (hash(read(binding.manifest)) !== binding.sha256) throw Error('manifest hash mismatch: ' + name);
  for (const stream of [data.stdout, data.stderr]) if (hash(fs.readFileSync(path.join(out, stream.path))) !== stream.sha256) throw Error('raw stream hash mismatch: ' + name);
  for (const entry of JSON.parse(read(data.controls.manifest)).filter(entry => /\.(mjs|md|sh|json|go)$/.test(entry.path))) {
    if (hash(fs.readFileSync(path.join(out, 'control-' + entry.sha256 + path.extname(entry.path)))) !== entry.sha256) throw Error('historical control preimage mismatch: ' + entry.path);
  }
  return {receipt: name, sha256: hash(read(name)), ...data};
}).sort((a, b) => a.started.localeCompare(b.started));
const required = name => {
  const result = receipts.find(receipt => receipt.name === name);
  if (!result) throw Error('missing receipt: ' + name);
  return result;
};
const final = required('preservation-ps-bound-final');
if (final.child_exit_code !== 0) throw Error('final proof failed');
for (const entry of JSON.parse(read(final.source.manifest))) if (entry.sha256 !== (fs.existsSync(path.join(repo, entry.path)) ? hash(fs.readFileSync(path.join(repo, entry.path))) : null)) throw Error('final source no longer frozen: ' + entry.path);
const jsonLines = name => read(name).split('\n').filter(Boolean).map(line => JSON.parse(line));
const writerErrors = name => jsonLines(name + '.stdout').filter(event => event.Output?.includes('stderr Write returned')).map(event => ({test: event.Test, exact_output: event.Output}));
const ps = spawnSync('/bin/ps', ['-eo', 'pid,ppid,stat,args'], {encoding: 'utf8'});
if (ps.status !== 0) throw Error(ps.stderr);
const live = ps.stdout.split('\n').filter(line => /^\s*\d+\s+\d+\s+\S+\s+(?:\S*\/)?(?:go\s+(?:test|vet|run|list)\b|golangci-lint\S*\s+run\b|make\s+.*(?:lint|validate)|\S*gomadtool\.test\b)/.test(line));
if (live.length) throw Error('source/cache lane still live: ' + live.join('\n'));
const packet = {
  task: 'fn-109.50', status: 'source-progress; task acceptance open; root review pending',
  admission_head: base, original_lint_filter: original,
  source_before_edit: required('controls-baseline').source, source_final: final.source,
  additive_fixture: JSON.parse(read(final.source.manifest)).find(entry => entry.path.endsWith('/generator_diagnostic_output_test.go')),
  final_proof_tools: final.tools, final_proof_controls: final.controls,
  handover: {path: 'handover.md', sha256: hash(read('handover.md'))},
  receipts, preservation: jsonLines('preservation-ps-bound-final.stdout'),
  offline_proxy_archive_proof: jsonLines('proxy-archive-proof.stdout'),
  actual_writer_errors: {before_edit: writerErrors('controls-baseline'), final: writerErrors('controls-final'), exact96b_overlay: writerErrors('baseline-overlay-controls')},
  environment_prerequisite: {
    original: 'GOPROXY=off prevents private scratch-modcache adapter downloads despite existing shared archives',
    approved_successor: 'explicit file:// proxy reuses existing pinned sprig v3.2.3/v3.3.0 archives; eight archive/metadata inputs plus proxy list and checksum controls frozen; no network/archive/pin/source edits',
    only_override: required('ordinary-package-final-offline-proxy').environment_overrides,
    candidate_baseline_failure_equivalence: true,
  },
  actual_analyzer_delta: {scoped: {before: 68, after: 63, exit_before: 1, exit_after: 1}, original_integrated: {before: 213, after: 208, exit_before: 2, exit_after: 2, packages: 55, errcheck_before: 158, errcheck_after: 153, exhaustive: 2, forbidigo: 9, staticcheck: 44}, resolved: 5, introduced: 0, changed_residuals: 0, normalization: 'aggregate counts only; no location normalization', original_integrated_errortype_reached: false, standalone_errortype_exit: 0},
  retained_failures: [
    {receipt: 'generator-inventory-before-receipt.json', cause: 'artifact inventory SyntaxError; corrected additive receipt retained'},
    {receipt: 'focus-existing-final-receipt.json', cause: 'existing task46 status assertion and intermittent existing refresh git discovery failure; driver stopped exit1'},
    {receipts: ['ordinary-package-final-receipt.json', 'ordinary-package-baseline-overlay-receipt.json'], cause: 'identical inherited task46 assertion plus private adapter modcache GOPROXY=off environment prerequisite'},
    {receipts: ['ordinary-package-final-offline-proxy-receipt.json', 'ordinary-package-baseline-offline-proxy-receipt.json'], cause: 'sole existing task46 compatibility-pack invalid status1 versus expected2; exact identical error bytes'},
  ],
  unproved_acceptance_owners: [
    {owner: 'root', obligation: 'fresh independent source/evidence review; lifecycle and any progress commit; formal review only on green required tree'},
    {owner: 'fn-109.46 / separate root-admitted correction', obligation: 'inherited status assertion prevents full ordinary/focused GREEN; no task50 authority to edit'},
    {owner: 'fn-109 original semantic/source owners and final consumer fn-109.21', obligation: 'original first-baseline, fixed-identity, preservation/predecessor/full/default/affected/functional, non-native measurements and formal source acceptance wherever unproved; 208 original-base lint residuals'},
    {owner: 'fn-113.2 environment prerequisite', obligation: 'retained initial offline adapter failure now resolved only with disclosed pinned file proxy, not native qualification'},
    {owner: 'fn-149.1/.2/.4 and fn-128 native qualification/CI', obligation: 'deferred and unverified; no native revival, pass, bound, PR/push/CI authority'},
  ],
  explicit_profile_skip: required('profile-check-final').observations,
  terminal_carriers: [{id: 89458, exit: 0}, {id: 32919, exit: 0}, {id: 37265, exit: 1}, {id: 55930, exit: 0}, {id: 7908, exit: 0}, {id: 93015, exit: 0}, {id: 93686, exit: 0}, {id: 83276, exit: 0}],
  additional_terminal_commands: ['first inventory immediate carrier exit1, failed child receipt retained', 'prepare-overlay foreground command exit0; exact saved source originals bound in all subsequent receipts'],
  live_go_build_lint_generator_children: live,
  final_seal: 'packet-seal-receipt.json produced after this write-once evidence file; its raw stdout binds this evidence hash',
  commits: [], prs: [], tests: receipts.map(receipt => receipt.command),
};
const bytes = JSON.stringify(packet, null, 2) + '\n';
fs.writeFileSync(path.join(out, 'evidence.json'), bytes, {flag: 'wx'});
console.log(JSON.stringify({evidence: 'evidence.json', sha256: hash(bytes), prior_receipts: receipts.length, source: final.source, proof_tools: final.tools, proof_controls: final.controls, handover: packet.handover, live_go_build_lint_generator_children: live}));
