import fs from 'node:fs';
import path from 'node:path';
import {repo, out, base, original, hash, git, sources} from './run.mjs';

const read = name => fs.readFileSync(path.join(out, name), 'utf8');
const receipt = name => JSON.parse(read(name + '-receipt.json'));
const events = name => read(name + '.stdout').trim().split('\n').filter(line => line.startsWith('{')).map(line => JSON.parse(line));
const predecessor = path.resolve(out, '../../task-50/generator-stderr-20261009');
const changed = 'tools/gomad3/cmd/gomadtool/maintainer_output_test.go';
const addition = 'tools/gomad3/cmd/gomadtool/usage_status_preservation_test.go';
const before = git('show', base + ':' + changed);
const anchor = '\tfor _, command := range []string{"build-key", "patch-validate", "script-validate", "patch-materialize", "patch-regenerate", "test", "toolchain-build", "boundary-generate", "compatibility-pack", "upgrade-dossier"} {\n';
const insert = '\t\tinvalidStatus := 2\n\t\tif command == "compatibility-pack" {\n\t\t\tinvalidStatus = 1\n\t\t}\n';
const row = '\t\t\t{"invalid", []string{command, "--not-a-flag"}, 2},';
if (before.split(anchor).length !== 2 || before.split(row).length !== 2) throw Error('ambiguous admitted fixture surface');
const expected = before.replace(anchor, anchor + insert).replace(row, row.replace(', 2}', ', invalidStatus}'));
const after = fs.readFileSync(path.join(repo, changed), 'utf8');
if (after !== expected) throw Error('existing fixture changed beyond exact admitted amendment');
const current = sources(), basis = JSON.parse(fs.readFileSync(receipt('fixture-red').source.basis));
const overrides = current.filter(entry => basis.find(old => old.path === entry.path)?.sha256 !== entry.sha256);
if (JSON.stringify(overrides.map(entry => entry.path).sort()) !== JSON.stringify([changed, addition].sort())) throw Error('unadmitted source change');
const originals = git('ls-tree', '-r', '--name-only', base, '--', 'tools/gomad3').trim().split('\n');
for (const relative of originals) if (relative !== changed && hash(fs.readFileSync(path.join(repo, relative))) !== hash(git('show', base + ':' + relative))) throw Error('unaffected original differs: ' + relative);
const additions = git('ls-files', '--cached', '--others', '--exclude-standard', '--', 'tools/gomad3').trim().split('\n').filter(relative => !originals.includes(relative));
if (JSON.stringify(additions) !== JSON.stringify([addition])) throw Error('unadmitted module addition');
console.log(JSON.stringify({existing_fixture: {path: changed, original_sha256: hash(before), final_sha256: hash(after), exact_admitted_insertions: insert, exact_admitted_row: row.replace(', 2}', ', invalidStatus}')}, unaffected_original_gomad3_files: originals.length - 1, source: receipt('fixture-green').source, production_and_permanent_task8_bytes_preserved: true}));
for (const name of ['preservation-before', 'preservation-after']) {
  const data = receipt(name);
  const fixture = data.source.overrides.find(entry => entry.path === addition);
  if (data.child_exit_code !== 0 || data.counts.pass !== 12 || data.counts.fail !== 0 || data.counts.skip !== 0 || fixture?.sha256 !== hash(fs.readFileSync(path.join(repo, addition)))) throw Error('preservation command changed or unproved');
  const observations = events(name).filter(event => event.Test?.includes('TestRunCompatibilityPackUsageStatusPreservation/') && event.Output?.includes('stderr-calls='));
  if (observations.length !== 4 || observations.some(event => !event.Output.includes('stderr-calls=1') || !event.Output.includes('stdout-calls=0') || !event.Output.includes('publication unchanged'))) throw Error('public counted preservation observations missing');
  console.log(JSON.stringify({preservation_receipt: name, child_exit_code: 0, authoritative_preservation_passes: 12, public_route_leaves: observations.map(event => ({test: event.Test, observation: event.Output}))}));
}
const red = receipt('fixture-red'), green = receipt('fixture-green');
if (red.command !== green.command || red.child_exit_code !== 1 || green.child_exit_code !== 0) throw Error('fixture RED/GREEN command identity differs');
const invalid = 'TestRunMaintainerOutputPrimaryFailures/compatibility-pack/invalid';
if (!events('fixture-red').some(event => event.Test === invalid && event.Output?.includes('primary status = 1, want 2')) || !events('fixture-green').some(event => event.Test === invalid && event.Action === 'pass')) throw Error('exact existing fixture transition missing');
console.log(JSON.stringify({existing_fixture_red_green: {base, red: 'fixture-red', green: 'fixture-green', case: invalid, red_diagnostic: 'primary status = 1, want 2'}, historical_task46: 'immutable referenced packet; no current-source inference', exact96b_red_reference: path.join(predecessor, 'baseline-overlay-inherited-focus-receipt.json')}));
const inputs = JSON.parse(read('mutant-inputs.json'));
for (const entry of [inputs.original, inputs.mutant, inputs.overlay]) if (hash(fs.readFileSync(entry.path)) !== entry.sha256) throw Error('scratch mutation input differs');
const source = fs.readFileSync(path.join(repo, inputs.production.path), 'utf8');
if (hash(source) !== inputs.production.sha256 || source.split(inputs.original_block).length !== 3 || source.split(inputs.original_block).join(inputs.replacement_block) !== fs.readFileSync(inputs.mutant.path, 'utf8')) throw Error('scratch mutation is not exact two returns');
if (JSON.stringify(inputs.replacement_occurrences.map(entry => entry.return_line)) !== JSON.stringify([25, 44])) throw Error('unexpected mutation lines');
for (const [name, test, leaves] of [
  ['mutant-permanent', 'TestCompatibilityPackSourceUsagePreservesBytesAndWriteFailure', ['no_command_closed_writer', 'unknown_command_closed_writer']],
  ['mutant-public', 'TestRunCompatibilityPackUsageStatusPreservation', ['missing_subcommand/EBADF_stderr', 'unknown_subcommand/EBADF_stderr']],
]) {
  const data = receipt(name), observed = events(name), wanted = leaves.map(leaf => test + '/' + leaf);
  const failures = observed.filter(event => event.Action === 'fail' && event.Test).map(event => event.Test);
  const terminalLeaves = failures.filter(failure => !failures.some(other => other.startsWith(failure + '/')));
  if (data.child_exit_code !== 1 || data.counts.skip !== 0 || JSON.stringify(terminalLeaves) !== JSON.stringify(wanted)) throw Error('unexpected mutant leaf failures');
  for (const leaf of wanted) {
    const output = observed.filter(event => event.Test === leaf && event.Action === 'output').map(event => event.Output).join('');
    if (!/status(?:\s*=\s*|=)2.*want(?:\s*=\s*|\s+)1/.test(output)) throw Error('mutant lacks literal failed-usage status rejection');
  }
  console.log(JSON.stringify({mutation_receipt: name, child_exit_code: 1, failed_terminal_leaves: terminalLeaves, reason: 'actual status 2 versus independently literal expected status 1'}));
}
if (receipt('mutation-proof').child_exit_code !== 1 || !read('mutation-proof.stderr').includes('mutation rejected on unexpected surface')) throw Error('historical audit failure not retained');
console.log(JSON.stringify({mutation_inputs: inputs, failed_outer_audit: {receipt: 'mutation-proof', cause: 'intermediate public subtest ancestors incorrectly included as leaves', original_control: 'mutation.mjs', sha256: hash(read('mutation.mjs'))}, corrected_audit: 'terminal failed leaf selection over existing receipts; zero repeated suites', production_unchanged: true}));
const diagnostics = data => {
  const result = [];
  for (const line of data.split('\n')) {
    if (/^tools\/gomad3\/[^:]+:\d+:\d+: /.test(line)) result.push([line]);
    else if (/^\s/.test(line) && result.length > 0) result.at(-1).push(line);
  }
  return result.map(lines => lines.join('\n'));
};
for (const [beforeName, afterName, priorName, count, exit] of [
  ['scoped-baseline', 'scoped-final', 'root-generator-scoped-lint', 63, 1],
  ['integrated-baseline', 'integrated-final', 'root-generator-integrated-lint', 208, 2],
]) {
  const a = diagnostics(read(beforeName + '.stdout')), b = diagnostics(read(afterName + '.stdout'));
  const prior = diagnostics(fs.readFileSync(path.join(predecessor, priorName + '.stdout'), 'utf8'));
  if (a.length !== count || b.length !== count || JSON.stringify(a) !== JSON.stringify(b) || JSON.stringify(b) !== JSON.stringify(prior) || receipt(afterName).child_exit_code !== exit) throw Error('lint residual message/source bytes changed');
  console.log(JSON.stringify({lint: afterName, original_baseline: original, actual_diagnostic_count: count, actual_exit: exit, introduced: 0, resolved: 0, changed_residual_message_source_bytes: 0, compared_records_sha256: hash(JSON.stringify(b)), predecessor_receipt: priorName + '-receipt.json', aggregate_green: false}));
}
const fast = receipt('fast-final');
if (fast.child_exit_code !== 0 || !read('fast-final.stdout').includes('55 host packages') || !read('fast-final.stdout').includes('0 issues.') || !read('fast-baseline.stdout').includes('No changed Go packages to lint.')) throw Error('fast gate coverage differs');
console.log(JSON.stringify({fast_final: {actual_exit: 0, host_packages: 55, admission_base_filter: base, emitted_findings: 0, errortype_reached: true}, fast_baseline: 'numeric exit 0, zero selected packages; not a coverage pass', original_base_integrated_errortype: 'unreached after analyzer RED', standalone_errortype: receipt('errortype-final').child_exit_code}));
for (const name of ['source-darwin-final', 'source-linux-final']) {
  const metadata = JSON.parse(read(name + '.stdout'));
  if (metadata.Error || metadata.DepsErrors?.length || metadata.ImportPath !== 'go.temporal.io/server/tools/gomad3/cmd/gomadtool' || !metadata.TestGoFiles.includes(path.basename(addition)) || !metadata.TestGoFiles.includes('maintainer_output_test.go')) throw Error('wrong or incomplete static source set');
  console.log(JSON.stringify({static_source_set: name, go_files: metadata.GoFiles.length, test_files: metadata.TestGoFiles.length, error: null, fixture_selected: true, metadata_sha256: hash(read(name + '.stdout')), native_execution: false}));
}
const inventoryPath = path.join(predecessor, 'generator-inventory-before-corrected.stdout');
const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8').split('\n')[0]);
for (const entry of [...inventory.generator_inputs, ...inventory.generated_outputs]) {
  const expected = basis.find(old => old.path === entry.path);
  if (!expected || hash(fs.readFileSync(path.join(repo, entry.path))) !== expected.sha256) throw Error('generator input/output changed');
}
console.log(JSON.stringify({generator_inventory_reference: inventoryPath, inventory_sha256: hash(fs.readFileSync(inventoryPath)), inputs: inventory.generator_inputs.length, outputs: inventory.generated_outputs.length, current_hash_basis: 'task50 final source manifest', input_and_generated_bytes_unchanged: true, check_only_validation_exit: receipt('validate-final').child_exit_code, generation: false}));
for (const [relative, expected] of [['.turbo/plans/gomad3-glossary-update.md', '97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188'], ['.turbo/technical-debt.md', 'c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287']]) {
  if (hash(fs.readFileSync(path.join(repo, relative))) !== expected) throw Error('unrelated user file changed');
  console.log(JSON.stringify({preserved_user_file: relative, sha256: expected}));
}
const names = fs.readdirSync(out).filter(name => name.endsWith('-receipt.json'));
for (const name of names) {
  const data = JSON.parse(read(name));
  if (!Number.isInteger(data.child_exit_code) || data.signal || data.error || !data.source_unchanged || !data.tools_unchanged || !data.controls_unchanged) throw Error('inconclusive receipt binding: ' + name);
  for (const stream of [data.stdout, data.stderr]) if (hash(fs.readFileSync(path.join(out, stream.path))) !== stream.sha256) throw Error('raw stream hash differs');
  for (const field of [data.source, data.tools]) if (hash(fs.readFileSync(field.basis)) !== field.basis_sha256) throw Error('predecessor manifest hash differs');
  for (const entry of data.controls) if (hash(fs.readFileSync(entry.path)) !== entry.sha256) throw Error('historical control no longer available');
  if (data.git_trace && hash(fs.readFileSync(data.git_trace.path)) !== data.git_trace.sha256) throw Error('Git trace differs');
}
console.log(JSON.stringify({audited_existing_receipts: names.length, audited_raw_streams: names.length * 2, historical_controls_unchanged: true, formal_review: 'root owns; required full package and original-base lint remain red', all_gate_commands_terminal: true}));
