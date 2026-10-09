import fs from 'node:fs';
import path from 'node:path';
import {repo, out, hash, run} from './run.mjs';

const relative = 'tools/gomad3/cmd/gomadtool/compatibility_pack.go';
const production = path.join(repo, relative);
const original = fs.readFileSync(production, 'utf8');
const old = '\t\tif _, err := fmt.Fprintln(stderr, "usage: gomadtool compatibility-pack discover|review|refresh|generate|check|qualify [flags]"); err != nil {\n\t\t\treturn 1\n\t\t}';
const replacement = old.replace('return 1', 'return 2');
if (original.split(old).length !== 3) throw Error('mutation requires exactly two usage failure returns');
const positions = [...original.matchAll(new RegExp(old.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'))].map(match => ({byte_offset: Buffer.byteLength(original.slice(0, match.index)), return_line: original.slice(0, match.index).split('\n').length + 1}));
const mutant = original.split(old).join(replacement);
const scratch = fs.mkdtempSync(path.join(repo, '.flow/tmp/task51-usage-mutant-'));
const originalPath = path.join(scratch, 'original.go'), mutantPath = path.join(scratch, 'mutant.go'), overlayPath = path.join(scratch, 'overlay.json');
fs.writeFileSync(originalPath, original, {flag: 'wx'});
fs.writeFileSync(mutantPath, mutant, {flag: 'wx'});
const overlay = JSON.stringify({Replace: {[production]: mutantPath}}) + '\n';
fs.writeFileSync(overlayPath, overlay, {flag: 'wx'});
const inputs = {production: {path: relative, sha256: hash(original)}, original: {path: originalPath, sha256: hash(original)},
  mutant: {path: mutantPath, sha256: hash(mutant)}, overlay: {path: overlayPath, sha256: hash(overlay)},
  replacement_occurrences: positions, original_block: old, replacement_block: replacement, replacement_count: 2};
fs.writeFileSync(path.join(out, 'mutant-inputs.json'), JSON.stringify(inputs, null, 2) + '\n', {flag: 'wx'});
console.log(JSON.stringify(inputs));
for (const [name, test] of [['mutant-permanent', 'TestCompatibilityPackSourceUsagePreservesBytesAndWriteFailure'], ['mutant-public', 'TestRunCompatibilityPackUsageStatusPreservation']]) {
  const receipt = run(name, "go -C tools/gomad3 test -json -tags test_dep -count=1 -overlay=" + overlayPath + " ./cmd/gomadtool -run '^" + test + "$'");
  const events = fs.readFileSync(path.join(out, name + '.stdout'), 'utf8').trim().split('\n').map(line => JSON.parse(line));
  const expectedLeaves = name === 'mutant-permanent' ? [test + '/no_command_closed_writer', test + '/unknown_command_closed_writer'] : [test + '/missing_subcommand/EBADF_stderr', test + '/unknown_subcommand/EBADF_stderr'];
  const failedLeaves = events.filter(event => event.Action === 'fail' && event.Test && event.Test !== test).map(event => event.Test);
  if (receipt.child_exit_code !== 1 || JSON.stringify(failedLeaves) !== JSON.stringify(expectedLeaves)) throw Error('mutation rejected on unexpected surface');
  for (const leaf of expectedLeaves) {
    const output = events.filter(event => event.Test === leaf && event.Action === 'output').map(event => event.Output).join('');
    if (!/status(?:\s*=\s*|=)2.*want(?:\s*=\s*|\s+)1/.test(output)) throw Error('mutation lacks expected status 2 versus literal 1');
  }
  console.log(JSON.stringify({receipt: name, child_exit_code: receipt.child_exit_code, rejected_leaves: failedLeaves, reason: 'actual status 2 versus independently literal expected status 1'}));
}
for (const entry of [inputs.original, inputs.mutant, inputs.overlay]) if (hash(fs.readFileSync(entry.path)) !== entry.sha256) throw Error('mutation input drift');
if (hash(fs.readFileSync(production)) !== inputs.production.sha256) throw Error('production changed during mutation');
console.log(JSON.stringify({production_unchanged: true, mutation_inputs_unchanged: true, scratch_only: true}));
