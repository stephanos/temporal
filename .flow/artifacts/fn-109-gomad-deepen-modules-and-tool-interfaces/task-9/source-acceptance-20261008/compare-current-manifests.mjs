import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

const out = path.dirname(new URL(import.meta.url).pathname);
const names = ['source-62a6deb0b7efdc144da8656a1fe3a7d3613a40c3766dcdbc59c9b8e959e38b2e.json', 'source-04bd8c60a587c585c936c62f8fc5a3eff797d6b1d518098d08f1adf70571b4e9.json'];
const inputs = names.map(name => {
  const bytes = fs.readFileSync(path.join(out, name));
  return {name, sha256: crypto.createHash('sha256').update(bytes).digest('hex'), entries: JSON.parse(bytes)};
});
const maps = inputs.map(input => new Map(input.entries.map(entry => [entry.path, entry.sha256])));
const differences = [...new Set(inputs.flatMap(input => input.entries.map(entry => entry.path)))].sort().filter(file => maps[0].get(file) !== maps[1].get(file)).map(file => ({path: file, before: maps[0].get(file), after: maps[1].get(file)}));
if (differences.length !== 1 || differences[0].path !== 'tools/gomad3/target/go_command_source_test.go') throw Error('unexpected source changes between gate candidates');
const result = {inputs: inputs.map(({name, sha256, entries}) => ({name, sha256, count: entries.length})), differences, scope: 'Only the authored QF1003 test-control switch edit changed. Production inputs of the earlier controls are unchanged; whole-source receipt identities remain different and are never relabeled current.'};
fs.writeFileSync(path.join(out, 'candidate-source-delta.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx'});
console.log(JSON.stringify(result));
