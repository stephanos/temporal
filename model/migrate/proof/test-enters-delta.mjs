import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';

const [before, after, scratch] = process.argv.slice(2);
fs.mkdirSync(scratch, {recursive:true});
const accepted = JSON.parse(fs.readFileSync(after));
const rule = model => model.functions.find(f => f.name === 'answerDispatch.rules.backoff').body.if.then.list.items[0].construct;
const mutations = {
  extraFact(model) { rule(model).args[2].list = {items:[{literal:{enum:{type:'temporal.features.activity.mini.protocol.AnswerFact',case:'acceptedStale'}}}]}; },
  extraStateWrite(model) { rule(model).args[1].copy.updates.push({name:'attempts',value:{literal:{int:'0'}}}); },
  rejectingOutcome(model) { rule(model).args[0].literal.enum.case = 'rejected'; },
  extraChoice(model) { const list = model.functions.find(f => f.name === 'answerDispatch.rules.backoff').body.if.then.list; list.items.push(structuredClone(list.items[0])); },
  changedTotal(model) { model.queries[0].total = '1'; },
  wrongSource(model) { model.actions[0].position.line += 1000; }
};
for (const [name, mutate] of Object.entries(mutations)) {
  const model = structuredClone(accepted);
  mutate(model);
  const file = path.join(scratch, `${name}.json`);
  fs.writeFileSync(file, JSON.stringify(model));
  const result = spawnSync(process.execPath, ['model/migrate/proof/check-enters-delta.mjs', before, file,
    path.join(scratch, `${name}.ledger.json`)], {encoding:'utf8', env:{...process.env,ENTERS_DIAGNOSTIC:'0'}});
  fs.writeFileSync(path.join(scratch, `${name}.log`), result.stdout + result.stderr);
  assert.equal(result.status, 1, name);
  assert.match(result.stderr, /AssertionError/, name);
}
console.log('PASS: six independent extra-fact/state/rejection/choice/total/source mutations are refused');
