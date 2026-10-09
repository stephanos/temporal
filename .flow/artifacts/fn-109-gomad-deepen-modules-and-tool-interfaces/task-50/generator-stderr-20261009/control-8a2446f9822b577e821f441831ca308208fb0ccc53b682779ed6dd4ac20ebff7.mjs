import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {repo, out, base, git, hash} from './run.mjs';

const read = name => fs.readFileSync(path.join(out, name), 'utf8');
const receipt = name => JSON.parse(read(name + '-receipt.json'));
const substitutions = new Map([
  ['qualification_manifest.go', [
    ['\t\tfmt.Fprintln(stderr, "qualification-manifest-generate requires --spec and --output")', '\t\tif _, writeErr := fmt.Fprintln(stderr, "qualification-manifest-generate requires --spec and --output"); writeErr != nil {\n\t\t\treturn 2\n\t\t}'],
    ['\t\tfmt.Fprintln(stderr, err)', '\t\tif _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {\n\t\t\treturn 1\n\t\t}'],
  ]],
  ...['protocol.go', 'version.go', 'boundary.go'].map(name => [name, [['\t\tfmt.Fprintln(stderr, err)', '\t\tif _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {\n\t\t\treturn 1\n\t\t}']]]),
]);
for (const [name, changes] of substitutions) {
  const relative = 'tools/gomad3/cmd/gomadtool/' + name;
  const before = git('show', base + ':' + relative);
  let expected = before;
  for (const [old, replacement] of changes) {
    if (expected.split(old).length !== 2) throw Error('ambiguous admitted statement: ' + name);
    expected = expected.replace(old, replacement);
  }
  const after = fs.readFileSync(path.join(repo, relative), 'utf8');
  if (after !== expected || read('baseline-' + name) !== before) throw Error('production scope or saved baseline differs: ' + name);
  console.log(JSON.stringify({path: relative, before_sha256: hash(before), after_sha256: hash(after), exact_admitted_statement_changes: changes.length, original_other_bytes_preserved: true}));
}
const originals = git('ls-tree', '-r', '--name-only', base, '--', 'tools/gomad3').trim().split('\n');
const allowed = new Set([...substitutions.keys()].map(name => 'tools/gomad3/cmd/gomadtool/' + name));
for (const relative of originals.filter(relative => !allowed.has(relative))) {
  if (hash(fs.readFileSync(path.join(repo, relative))) !== hash(git('show', base + ':' + relative))) throw Error('unaffected original path differs: ' + relative);
}
const protectedAtAdmission = originals.filter(relative => relative.endsWith('_test.go') || /\/(startup_sources\.go|memory_sources\.go|go\.mod|go\.sum)$/.test(relative));
const historical401 = git('ls-tree', '-r', '--name-only', '21b30b4604788c9e2e54b6c33f3e47045725daf9', '--', 'tools/gomad3').trim().split('\n')
  .filter(relative => relative.endsWith('_test.go') || /\/(startup_sources\.go|memory_sources\.go|go\.mod|go\.sum)$/.test(relative));
if (historical401.length !== 401) throw Error('historical protected inventory differs');
for (const relative of historical401) if (hash(fs.readFileSync(path.join(repo, relative))) !== hash(git('show', '21b30b4604788c9e2e54b6c33f3e47045725daf9:' + relative))) throw Error('historical original fixture/pin/module differs: ' + relative);
const fixture = 'tools/gomad3/cmd/gomadtool/generator_diagnostic_output_test.go';
const additions = git('ls-files', '--cached', '--others', '--exclude-standard', '--', 'tools/gomad3').trim().split('\n').filter(relative => !originals.includes(relative));
if (JSON.stringify(additions) !== JSON.stringify([fixture])) throw Error('unadmitted source addition');
const baseline = receipt('controls-baseline'), final = receipt('controls-final');
const baselineSources = JSON.parse(read(baseline.source.manifest)), finalSources = JSON.parse(read(final.source.manifest));
const frozenFixture = finalSources.find(entry => entry.path === fixture);
if (!frozenFixture || hash(fs.readFileSync(path.join(repo, fixture))) !== frozenFixture.sha256 || baselineSources.find(entry => entry.path === fixture)?.sha256 !== frozenFixture.sha256) throw Error('additive controls changed across preservation runs');
for (const entry of finalSources) if (!allowed.has(entry.path) && entry.path !== fixture && entry.sha256 !== baselineSources.find(original => original.path === entry.path)?.sha256) throw Error('source manifest residual changed: ' + entry.path);
console.log(JSON.stringify({unaffected_original_gomad3_files: originals.length - allowed.size, protected_at_admission: protectedAtAdmission.length, historical_originals: historical401.length, selection: 'baseline git ls-tree, never mutable index', additive_fixture: frozenFixture}));
const inventory = JSON.parse(read('generator-inventory-before-corrected.stdout').split('\n')[0]);
for (const entry of [...inventory.generator_inputs, ...inventory.generated_outputs]) if (!allowed.has(entry.path) && hash(fs.readFileSync(path.join(repo, entry.path))) !== entry.sha256) throw Error('generator input/output differs: ' + entry.path);
console.log(JSON.stringify({generator_inputs: inventory.generator_inputs.length, generated_outputs: inventory.generated_outputs.length, generated_bytes_unchanged: true, admitted_generator_command_input_changes: [...allowed], actual_generation: false}));
for (const line of read('generator-inventory-before-corrected.stdout').trim().split('\n').slice(1)) {
  const entry = JSON.parse(line);
  if (hash(fs.readFileSync(path.join(repo, entry.user_untracked))) !== entry.sha256) throw Error('user file changed: ' + entry.user_untracked);
}
const sites = [['boundary.go', 55], ['protocol.go', 20], ['qualification_manifest.go', 22], ['qualification_manifest.go', 26], ['version.go', 20]];
for (const [beforeName, afterName, total, errcheck] of [['scoped-analyzer-red', 'scoped-analyzer-final', 68, 68], ['integrated-original-before', 'integrated-original-final', 213, 158]]) {
  const before = read(beforeName + '.stdout'), after = read(afterName + '.stdout');
  let expected = before;
  const removed = [];
  for (const [name, line] of sites) {
    const statement = name === 'qualification_manifest.go' && line === 22 ? '\t\tfmt.Fprintln(stderr, "qualification-manifest-generate requires --spec and --output")' : '\t\tfmt.Fprintln(stderr, err)';
    const block = 'tools/gomad3/cmd/gomadtool/' + name + ':' + line + ':15: Error return value of `fmt.Fprintln` is not checked (errcheck)\n' + statement + '\n\t\t            ^\n';
    if (expected.split(block).length !== 2) throw Error('missing exact admitted analyzer block: ' + name + ':' + line);
    expected = expected.replace(block, '');
    removed.push(block);
  }
  expected = expected.replace(total + ' issues:\n', (total - 5) + ' issues:\n').replace('* errcheck: ' + errcheck + '\n', '* errcheck: ' + (errcheck - 5) + '\n');
  if (expected !== after) throw Error('residual analyzer stdout changed: ' + afterName);
  console.log(JSON.stringify({before: beforeName, after: afterName, actual_before: total, actual_after: total - 5, resolved: removed, introduced: 0, changed_residuals: 0, residual_stdout_byte_identical: true, normalization: 'aggregate counts only; no location normalization needed', before_sha256: hash(before), after_sha256: hash(after)}));
}
const historicalIntegrated = fs.readFileSync(path.join(repo, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/digest-lint-20261009/digest-integrated-after.stdout'));
if (!historicalIntegrated.equals(Buffer.from(read('integrated-original-before.stdout')))) throw Error('fresh213 baseline differs from retained213 raw report');
console.log(JSON.stringify({retained_original_213_report_reproduced_byte_exactly: true, original_filter: '951c5516e9e7b3066e7e069adda9565cfd68844c', later_22_filter: 'different historical scope, no new aggregate claim'}));
const events = name => read(name + '.stdout').trim().split('\n').map(line => JSON.parse(line));
for (const suffix of ['', '-offline-proxy']) {
  const finalName = 'ordinary-package-final' + suffix;
  const baselineName = suffix ? 'ordinary-package-baseline-offline-proxy' : 'ordinary-package-baseline-overlay';
  const failures = name => events(name).filter(event => event.Action === 'fail' && event.Test).map(event => event.Test);
  const errors = name => events(name).filter(event => event.OutputType === 'error').map(event => ({test: event.Test, output: event.Output}));
  if (JSON.stringify(failures(finalName)) !== JSON.stringify(failures(baselineName)) || JSON.stringify(errors(finalName)) !== JSON.stringify(errors(baselineName))) throw Error('ordinary package baseline/final failure differs');
  console.log(JSON.stringify({final: finalName, baseline: baselineName, identical_failures: failures(finalName), identical_failure_diagnostics: errors(finalName), counts: receipt(finalName).counts, top_level_counts: receipt(finalName).top_level_counts, full_package_green: false}));
}
for (const name of fs.readdirSync(out).filter(name => name.endsWith('-receipt.json'))) {
  const data = JSON.parse(read(name));
  if (!Number.isInteger(data.child_exit_code) || data.signal || data.error || !data.source_unchanged || !data.tools_unchanged || !data.controls_unchanged) throw Error('nonterminal or changed gate binding: ' + name);
  for (const binding of [data.source, data.tools, data.controls]) if (hash(read(binding.manifest)) !== binding.sha256) throw Error('receipt manifest hash differs: ' + name);
  for (const stream of [data.stdout, data.stderr]) if (hash(fs.readFileSync(path.join(out, stream.path))) !== stream.sha256) throw Error('raw stream hash differs: ' + name);
  for (const entry of JSON.parse(read(data.controls.manifest)).filter(entry => /\.(mjs|md|sh|json|go)$/.test(entry.path))) {
    const archived = path.join(out, 'control-' + entry.sha256 + path.extname(entry.path));
    if (hash(fs.readFileSync(archived)) !== entry.sha256) throw Error('historical control preimage differs: ' + entry.path);
  }
}
console.log(JSON.stringify({all_existing_raw_receipts_and_hashed_control_preimages_preserved: true, independent_source_progress_review: 'root pending', formal_review: 'unproved; required tree red'}));
const processList = spawnSync('/bin/ps', ['-eo', 'pid,ppid,stat,args'], {encoding: 'utf8'});
if (processList.status !== 0) throw Error(processList.stderr);
const live = processList.stdout.split('\n').filter(line => /^\s*\d+\s+\d+\s+\S+\s+(?:\S*\/)?(?:go\s+(?:test|vet|run|list)\b|golangci-lint\S*\s+run\b|make\s+.*(?:lint|validate)|\S*gomadtool\.test\b)/.test(line));
if (live.length !== 0) throw Error('Go/build/lint/generator command still live: ' + live.join('\n'));
console.log(JSON.stringify({live_go_build_lint_generator_children: live, execution_lane: 'proof child is foreground; all Go/cache gate children terminal'}));
