import fs from 'node:fs';
import path from 'node:path';
import {repo, out, base, hash, git, sources} from './run.mjs';

const files = fs.readdirSync(out).filter(name => name.endsWith('-receipt.json')).sort();
const receipts = files.map(name => ({name, sha256: hash(fs.readFileSync(path.join(out, name))), data: JSON.parse(fs.readFileSync(path.join(out, name)))}));
for (const entry of receipts) {
  if (!Number.isInteger(entry.data.child_exit_code) || entry.data.signal || entry.data.error || !entry.data.source_unchanged || !entry.data.tools_unchanged || !entry.data.controls_unchanged) throw Error('inconclusive receipt');
  for (const stream of [entry.data.stdout, entry.data.stderr]) if (hash(fs.readFileSync(path.join(out, stream.path))) !== stream.sha256) throw Error('raw output changed');
}
const final = receipts.find(entry => entry.data.name === 'source-proof').data;
if (final.child_exit_code !== 0 || final.source.inventory_sha256 !== hash(JSON.stringify(sources()))) throw Error('source proof no longer matches candidate');
const evidence = {task_id: 'fn-109-gomad-deepen-modules-and-tool-interfaces.51', status: 'in_progress', worker_terminal: true,
  base_commit: base, head: git('rev-parse', 'HEAD').trim(), commits: git('rev-list', '--reverse', base + '..HEAD').trim().split('\n').filter(Boolean), prs: [],
  tests: receipts.map(entry => entry.data.command), source: final.source, tool_basis: final.tools.basis, tool_basis_sha256: final.tools.basis_sha256,
  summary: {path: path.join(out, 'handover.md'), sha256: hash(fs.readFileSync(path.join(out, 'handover.md')))},
  baseline: 'red (actual pre-edit compatibility-pack invalid fixture status 1 versus expected 2; explicitly admitted correction)',
  tier: 'session (jev-unavailable(no_key)); project implementer requested gpt-6.1-sol/high; actual telemetry unobserved',
  receipts: receipts.map(entry => ({path: entry.name, sha256: entry.sha256, command: entry.data.command, exit: entry.data.child_exit_code, elapsed_seconds: entry.data.elapsed_seconds,
    source_inventory_sha256: entry.data.source.inventory_sha256, counts: entry.data.counts, top_level_counts: entry.data.top_level_counts, stdout: entry.data.stdout, stderr: entry.data.stderr,
    ...(entry.data.git_trace ? {git_trace: entry.data.git_trace} : {})})),
  preservation_proof: {receipt: 'source-proof-receipt.json', stdout_sha256: final.stdout.sha256},
  mutation: {inputs: 'mutant-inputs.json', sha256: hash(fs.readFileSync(path.join(out, 'mutant-inputs.json'))), rejected: ['mutant-permanent', 'mutant-public'],
    retained_audit_failure: 'mutation-proof', corrected_audit: 'source-proof; terminal leaf selection, no repeated tests'},
  limitations: ['ordinary-package-final exit1 from independent unselected_output Git init0/add128 recognition failure; cause unproved',
    'scoped unfiltered63 exit1 and original-base configured208 exit2 unchanged; integrated errortype unreached',
    'baseline fast exit0 selected zero packages; no coverage green', 'root independent source-progress review pending; no worker verdict, lifecycle mutation or commit',
    'original retained source obligations stay open wherever unproved; native fn128/fn149 deferred/unverified'],
  terminal: {live_child_handles: [], final_gate_command_ownership: 'all gate children returned numeric statuses before handover; packet seal itself returns afterward'}};
fs.writeFileSync(path.join(out, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n', {flag: 'wx'});
console.log(JSON.stringify({receipts: receipts.length, raw_streams: receipts.length * 2, evidence: {path: path.join(out, 'evidence.json'), sha256: hash(fs.readFileSync(path.join(out, 'evidence.json')))}, summary: evidence.summary,
  source: evidence.source, commits: evidence.commits, status: evidence.status, source_and_tool_unchanged: true, seal_receipt: 'packet-seal-receipt.json (additional, outside evidence to avoid self-reference)'}));
