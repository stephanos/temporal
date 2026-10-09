import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(new URL(import.meta.url).pathname);
const prior = path.join(path.dirname(out), 'source-acceptance-20261008');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const entry = file => ({path: file, sha256: hash(fs.readFileSync(file))});
const git = (...args) => {
  const result = spawnSync('git', args, {cwd: repo});
  if (result.status !== 0) throw Error(result.stderr.toString());
  return result.stdout;
};
const commands = [];
for (const [name, handle] of [
  ['resume-overflow-red', 71366], ['resume-overflow-green', 37654], ['resume-overflow-focused', 10284],
  ['resume-overflow-preservation', 74813], ['resume-overflow-preservation-final', 29958],
  ['resume-overflow-architecture', 37247], ['resume-overflow-generated', 59070], ['resume-overflow-errortype', 4633],
  ['resume-overflow-lint-fast', 34235], ['resume-overflow-lint-scoped', 32388], ['resume-overflow-lint-integrated', 17490],
  ['resume-overflow-source-darwin', 34883], ['resume-overflow-source-linux', 61209], ['resume-overflow-format', 58689],
  ['resume-upgrade-tmp-diagnostic', 63849], ['resume-upgrade-workspace-diagnostic', 23136],
]) {
  const receipt = path.join(prior, name + '-receipt.json');
  const data = JSON.parse(fs.readFileSync(receipt));
  if (!data.source_unchanged || !data.tools_unchanged || !data.controls_unchanged || data.signal || data.error || data.exit_code == null) throw Error('incomplete gate ' + name);
  for (const stream of ['stdout', 'stderr']) {
    if (hash(fs.readFileSync(path.join(prior, data[stream].path))) !== data[stream].sha256) throw Error('changed raw stream ' + name);
  }
  for (const input of ['source', 'tools', 'controls']) {
    if (hash(fs.readFileSync(path.join(prior, data[input].manifest))) !== data[input].sha256) throw Error('changed input manifest ' + name);
  }
  const actualDiagnosticEnvironment = name.startsWith('resume-upgrade-') ? JSON.parse(fs.readFileSync(path.join(prior, data.stdout.path), 'utf8').split('\n')[0]) : null;
  commands.push({name, receipt: entry(receipt), unified_exec_session: handle, terminal: true, exit_code: data.exit_code,
    elapsed_seconds: data.elapsed_seconds, command: data.command, counts: data.counts, source: data.source,
    actual_diagnostic_environment: actualDiagnosticEnvironment});
}
const originalTests = git('ls-files', 'tools/gomad3').toString().trim().split('\n').filter(file => file.endsWith('_test.go'));
for (const file of originalTests) {
  if (hash(git('show', 'HEAD:' + file)) !== hash(fs.readFileSync(path.join(repo, file)))) throw Error('original test changed ' + file);
}
const modified = git('diff', '--name-only', 'HEAD', '--', 'tools/gomad3').toString().trim().split('\n');
if (JSON.stringify(modified) !== JSON.stringify(['tools/gomad3/target/capability_collection.go'])) throw Error('unexpected source modification');
const green = commands.find(item => item.name === 'resume-overflow-green');
const finalSource = JSON.parse(fs.readFileSync(path.join(prior, green.source.manifest)));
for (const input of finalSource) {
  if (hash(fs.readFileSync(path.join(repo, input.path))) !== input.sha256) throw Error('final source changed ' + input.path);
}
const evidence = {
  task: 'fn-109-gomad-deepen-modules-and-tool-interfaces.9', head: git('rev-parse', 'HEAD').toString().trim(), commits: [], prs: [],
  implementation: 'Public collectCapabilityListing translates only direct typed overflow after InvalidInput classification; private ListWith and command mechanism unchanged.',
  model: {requested: 'gpt-6.1-sol/high', tier: 'session (jev-unavailable(no_key))', actual: 'No executed-model metadata exposed'},
  source: green.source, final_source_rechecked: true, original_tracked_tests_unchanged: originalTests.length,
  source_files: ['tools/gomad3/target/capability_collection.go', 'tools/gomad3/target/capability_collection_overflow_test.go'].map(file => entry(path.join(repo, file))),
  admission: entry(path.join(out, 'admission.md')), commands,
  preservation: [entry(path.join(out, 'preservation.mjs')), entry(path.join(out, 'preservation-inputs.json')), entry(path.join(out, 'preservation-comparison.json'))],
  tests: commands.map(item => item.command),
  preparation_observation: 'Copy process PID 3167 is terminal and its complete manifest was verified before and after the final preservation command. Its exec session/exit was not retained because the tool result was projected to output only. The premature ENOENT preservation receipt remains inconclusive.',
  remaining: [
    'Root owns fresh independent source/evidence review, commits and lifecycle. Task9 remains in_progress; task40 predecessor acceptance remains open.',
    'Scoped lint retains two exact unchanged ST1005 diagnostics; integrated lint retains 24 exact unchanged diagnostics and never reaches integrated errortype.',
    'Original resume-defaults gate remains RED (86 named passes, five named failures); this repair does not alter hostexec default tests or claim aggregate default acceptance.',
    'Historical upgrade publication RED remains retained. Unchanged relevant 31-case selection passes on two fresh diagnostic roots; original failing-root behavior is not causally explained.',
    'Original retained acceptance and other source-owner requirements remain open wherever unproved. Native fn128/fn149 gates remain deferred and unverified; no native, CI, push or PR action occurred.',
  ],
};
fs.writeFileSync(path.join(out, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n', {flag: 'wx'});
console.log(JSON.stringify({commands: commands.length, source: evidence.source.sha256, unchanged_tests: originalTests.length, evidence: entry(path.join(out, 'evidence.json'))}));
