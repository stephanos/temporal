import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.dirname(new URL(import.meta.url).pathname);
const prior = path.join(path.dirname(out), 'source-acceptance-20261008');
const manifestPath = path.join(out, 'preservation-inputs.json');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const entry = file => ({path: file, sha256: hash(fs.readFileSync(file))});
const git = (...args) => {
  const result = spawnSync('git', args, {cwd: repo, encoding: 'utf8'});
  if (result.status !== 0) throw Error(result.stderr);
  return result.stdout;
};
const mode = process.argv[2];

if (mode === 'prepare') {
  if (fs.existsSync(manifestPath)) throw Error('preservation manifest already exists');
  const root = fs.mkdtempSync('/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task9-resume-preservation-');
  const graph = path.join(root, 'current');
  const relativePaths = [...new Set((git('ls-files', 'tools/gomad3', 'tools/gomad3sim', 'go.mod', 'go.sum') + git('ls-files', '--others', '--exclude-standard', 'tools/gomad3')).trim().split('\n'))].sort();
  const files = [];
  for (const relative of relativePaths) {
    const source = path.join(repo, relative);
    if (!fs.existsSync(source)) continue;
    const file = path.join(graph, relative);
    fs.mkdirSync(path.dirname(file), {recursive: true});
    fs.copyFileSync(source, file);
    files.push({...entry(file), source});
  }
  const fixture = path.join(prior, 'listing-control_test.go.txt');
  const added = path.join(graph, 'tools/gomad3/target/task9_listing_control_test.go');
  fs.writeFileSync(added, fs.readFileSync(fixture), {flag: 'wx'});
  files.push({...entry(added), source: fixture});
  const old = JSON.parse(fs.readFileSync(path.join(prior, 'listing-control-v2-inputs.json')));
  const controls = [entry(new URL(import.meta.url).pathname), entry(fixture), entry(path.join(prior, 'listing-control-v2-inputs.json')),
    ...old.files.filter(item => item.path.startsWith(old.scratch + '/')).map(item => entry(item.path)),
    ...['v2-listing-boundary-original.stdout', 'v2-listing-boundary-original-receipt.json', 'frozen-listing-current.stdout', 'frozen-listing-current-receipt.json'].map(name => entry(path.join(prior, name)))];
  const manifest = {head: git('rev-parse', 'HEAD').trim(), graph, caller_control: old.scratch, files, controls,
    scope: 'Complete copied current source graph; unchanged original listing test and literal shared caller/control paths. No overlay or normalization.'};
  fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + '\n', {flag: 'wx'});
  console.log(JSON.stringify({graph, files: files.length, controls: controls.length, manifest: entry(manifestPath)}));
} else if (mode === 'run') {
  const manifest = JSON.parse(fs.readFileSync(manifestPath));
  const verify = () => {
    for (const input of [...manifest.files, ...manifest.controls]) {
      if (hash(fs.readFileSync(input.path)) !== input.sha256) throw Error('input changed: ' + input.path);
    }
  };
  verify();
  const commandLog = path.join(manifest.caller_control, 'commands.jsonl');
  const beforeCommands = fs.readFileSync(commandLog);
  const result = spawnSync('go', ['test', '-json', '-count=1', '-tags', 'test_dep', './target', '-run', '^TestTask9ListingCharacterization$'], {
    cwd: path.join(manifest.graph, 'tools/gomad3'), env: {...process.env, TASK9_LISTING_CONTROL: manifest.caller_control}, encoding: 'utf8', maxBuffer: 16 * 1024 * 1024,
  });
  process.stdout.write(result.stdout ?? '');
  process.stderr.write(result.stderr ?? '');
  verify();
  if (result.status !== 0) throw Error('preservation test exit ' + result.status + ': ' + result.error);
  const rows = contents => contents.trim().split('\n').flatMap(line => {
    const event = JSON.parse(line);
    return event.Action === 'output' && event.Output.startsWith('TASK9_ROW ') ? [JSON.parse(event.Output.slice(10))] : [];
  });
  const original = rows(fs.readFileSync(path.join(prior, 'v2-listing-boundary-original.stdout'), 'utf8'));
  const frozen = rows(fs.readFileSync(path.join(prior, 'frozen-listing-current.stdout'), 'utf8'));
  const current = rows(result.stdout);
  if (original.length !== 22 || current.length !== 22 || frozen.length !== 22) throw Error('incomplete characterization');
  const allDifferences = original.flatMap((row, index) => JSON.stringify(row) === JSON.stringify(current[index]) ? [] : [{name: row.Name, original: row, current: current[index]}]);
  const ordinaryDifferences = allDifferences.filter(row => !row.name.endsWith('-overflow'));
  const publicDifferences = allDifferences.filter(row => row.name.startsWith('public-review/'));
  const standardOverflowChanges = current.flatMap((row, index) => row.Name.startsWith('private-standard/') && row.Name.endsWith('-overflow') && JSON.stringify(row) !== JSON.stringify(frozen[index]) ? [{name: row.Name, frozen: frozen[index], current: row}] : []);
  const commandsAfter = fs.readFileSync(commandLog);
  if (!commandsAfter.subarray(0, beforeCommands.length).equals(beforeCommands)) throw Error('existing command observations changed');
  const commands = commandsAfter.subarray(beforeCommands.length).toString().trim().split('\n').filter(Boolean).map(JSON.parse);
  const reaping = commands.map(command => {
    let gone = false;
    try { process.kill(command.pid, 0); } catch (error) { if (error.code !== 'ESRCH') throw error; gone = true; }
    return {...command, gone};
  });
  const report = {manifest: entry(manifestPath), original_rows: original, current_rows: current, normalization: 'None', public_cases: 11, ordinary_cases: 18,
    public_differences: publicDifferences, ordinary_differences: ordinaryDifferences, all_original_differences: allDifferences,
    standard_overflow_changes_from_frozen: standardOverflowChanges, command_log: {path: commandLog, before_sha256: hash(beforeCommands), after_sha256: hash(commandsAfter), new_commands: reaping},
    input_verification_before_and_after: true};
  fs.writeFileSync(path.join(out, 'preservation-comparison.json'), JSON.stringify(report, null, 2) + '\n', {flag: 'wx'});
  console.log(JSON.stringify({public_differences: publicDifferences.length, ordinary_differences: ordinaryDifferences.length,
    standard_overflow_changes: standardOverflowChanges.length, new_children: reaping.length, all_gone: reaping.every(row => row.gone)}));
  if (publicDifferences.length || ordinaryDifferences.length || standardOverflowChanges.length || !reaping.every(row => row.gone)) process.exitCode = 1;
} else {
  throw Error('preservation.mjs prepare|run');
}
