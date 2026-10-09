import fs from 'node:fs';
import path from 'node:path';
import {out, hash} from './run.mjs';

const read = name => fs.readFileSync(path.join(out, name));
if (process.argv[2] === 'archive') {
  for (const [name, archived] of [['handover.md', 'handover-initial.md'], ['evidence.json', 'evidence-initial.json']]) fs.writeFileSync(path.join(out, archived), read(name), {flag: 'wx'});
} else {
  const initial = read('handover-initial.md').toString();
  const current = read('handover.md').toString();
  if (initial.split('1,036 unaffected original Gomad files').length !== 2 || initial.split('../../../../../../tmp/').length !== 3 || current !== initial.replace('1,036 unaffected original Gomad files', '1,037 unaffected original Gomad files').replaceAll('../../../../../../tmp/', '../../../../tmp/')) throw Error('unadmitted handover correction');
  const evidence = JSON.parse(read('evidence-initial.json'));
  if (evidence.summary.sha256 !== hash(initial)) throw Error('initial handover binding differs');
  evidence.summary.sha256 = hash(current);
  fs.writeFileSync(path.join(out, 'evidence.json'), JSON.stringify(evidence, null, 2) + '\n');
  console.log(JSON.stringify({original_summary: {path: 'handover-initial.md', sha256: hash(initial)}, original_evidence: {path: 'evidence-initial.json', sha256: hash(read('evidence-initial.json'))},
    corrected_summary: {path: 'handover.md', sha256: hash(current)}, corrected_evidence: {path: 'evidence.json', sha256: hash(read('evidence.json'))},
    exact_changes: 'one count typo1036 to1037 and two relative .flow/tmp links; evidence changes only summary hash', source_changed: false, repeated_suites: 0}));
}
