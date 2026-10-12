import fs from 'node:fs';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';

const [beforePath, afterPath, ledgerPath] = process.argv.slice(2);
const beforeBytes = fs.readFileSync(beforePath), afterBytes = fs.readFileSync(afterPath);
const before = JSON.parse(beforeBytes), after = JSON.parse(afterBytes);
const prefix = 'temporal.features.activity.mini.protocol.Answering.answering$.effects$.';
const removed = [prefix + 'backOff', prefix + 'resume'];
const functions = new Map(before.functions.map(f => [f.name, f]));
const files = {
  answering: 'model/temporal/features/activity_mini/protocol/Answering.scala',
  dispatch: 'model/temporal/features/activity_mini/protocol/AnswerDispatch.scala'
};
const linePairs = {
  answering: {17:[19],18:[20],19:[21],20:[22],21:[23,24],23:[27,28,29],24:[30],25:[31],
    26:[32],27:[33,34],28:[35],29:[36],30:[37],31:[67],32:[70],33:[38],34:[39],35:[41,42],
    36:[43,44,45],37:[46],38:[47,48],39:[49],40:[50,51],41:[52],42:[53],43:[54],45:[57],
    46:[58],48:[60],49:[61],51:[63],52:[64],54:[66],55:[67],57:[69],58:[70],60:[72],61:[73],
    63:[75],64:[76],66:[78],67:[79]},
  dispatch: {13:[17],14:[18],17:[22],18:[23],23:[30,31],24:[32],25:[33,34],27:[38],28:[39],
    29:[41],31:[44],33:[52],34:[53],35:[58],36:[60],37:[62],38:[63],39:[68,69]}
};
const changes = [], observed = { answering: {}, dispatch: {} };
function position(a, b, path, inline = false) {
  if (a.file !== b.file) {
    assert.ok(inline && a.file === files.answering && [31, 32].includes(a.line) &&
      b.file === files.dispatch && b.line === 14 &&
      /construct\/args\/1\/(?:copy\/base\/)?position$/.test(path), path);
    changes.push({path, kind: 'declared-writer-position', before: a, after: b});
    return;
  }
  assert.deepEqual(Object.keys(a), Object.keys(b), path);
  if (a.line === b.line && !inline) return;
  const key = Object.keys(files).find(k => files[k] === a.file);
  assert.ok(key, `unexpected position file: ${path}`);
  (observed[key][a.line] ??= new Set()).add(b.line);
  changes.push({ path, kind: 'position', before: a, after: b });
  if (process.env.ENTERS_DIAGNOSTIC !== '1') {
    assert.ok(linePairs[key][a.line]?.includes(b.line), `unaccounted source coordinate ${key}:${a.line}->${b.line} at ${path}`);
  }
}
function compare(a, b, path, inline = false) {
  if (a && b && typeof a === 'object' && typeof b === 'object') {
    assert.deepEqual(Object.keys(a), Object.keys(b), `shape at ${path}`);
    for (const key of Object.keys(a)) {
      if (key === 'position') position(a[key], b[key], path + '/position', inline);
      else compare(a[key], b[key], path + '/' + key, inline);
    }
  } else assert.deepEqual(a, b, `value at ${path}`);
}
assert.deepEqual(Object.keys(before), Object.keys(after));
assert.equal(before.source, after.source);
assert.deepEqual(before.functions.filter(f => !after.functions.some(g => g.name === f.name)).map(f => f.name), removed);
const inlinings = { 'answerDispatch.rules.delayAnswer': removed[0], 'answerDispatch.rules.backoff': removed[1] };
for (const section of Object.keys(before).filter(k => k !== 'source')) {
  const left = section === 'functions' ? before[section].filter(f => !removed.includes(f.name)) : before[section];
  assert.equal(left.length, after[section].length, section);
  left.forEach((a, i) => {
    const b = after[section][i];
    const path = `${section}/${i}/${a.name ?? a.id ?? ''}`;
    if (section === 'functions' && inlinings[a.name]) {
      assert.equal(a.body.if.then.call.function, inlinings[a.name]);
      assert.deepEqual(a.body.if.then.call.args.map(x => x.var), ['s']);
      const core = functions.get(inlinings[a.name]);
      const expectedPhase = a.name.endsWith('delayAnswer') ? 'backingOff' : 'waiting';
      const step = b.body.if.then.list.items[0].construct;
      assert.equal(b.body.if.then.list.items.length, 1);
      assert.equal(step.type, 'framework.Step');
      assert.deepEqual(step.args[0].literal.enum, {type: 'framework.outcomes.Outcome', case: 'accepted'});
      assert.equal(step.args[1].copy.base.var, 's');
      assert.equal(step.args[1].copy.updates.length, 1);
      assert.equal(step.args[1].copy.updates[0].name, 'phase');
      assert.deepEqual(step.args[1].copy.updates[0].value.literal.enum,
        {type: 'temporal.features.activity.mini.protocol.AnswerPhase', case: expectedPhase});
      assert.deepEqual(step.args[2].list, {});
      assert.equal(step.args[3].literal.text, '');
      compare(core.body, b.body.if.then, path + '/inlined-core', true);
      const oldBody = {...a.body, if: {...a.body.if, then: b.body.if.then}};
      compare({...a, body: oldBody}, b, path);
      changes.push({path: path + '/body/if/then', kind: 'inline', function: core.name,
        before: a.body.if.then, after: b.body.if.then});
    } else compare(a, b, path);
  });
}
const coordinates = Object.fromEntries(Object.entries(observed).map(([file, pairs]) => [file,
  Object.fromEntries(Object.entries(pairs).map(([line, values]) => [line, [...values].sort((a,b)=>a-b)]))]));
const ledger = {beforeSha256: crypto.createHash('sha256').update(beforeBytes).digest('hex'),
  afterSha256: crypto.createHash('sha256').update(afterBytes).digest('hex'),
  removed: removed.map(name => ({name, declaration: functions.get(name), sha256:
    crypto.createHash('sha256').update(JSON.stringify(functions.get(name))).digest('hex')})),
  sections: Object.fromEntries(Object.entries(after).filter(([,v]) => Array.isArray(v)).map(([k,v])=>[k,v.length])),
  queries: after.queries.map(q => ({name: q.name, machine: q.scenario?.machine,
    limits: q.limits, total: q.total})), coordinates, changes};
fs.writeFileSync(ledgerPath, JSON.stringify(ledger, null, 2) + '\n');
console.log(JSON.stringify(coordinates, null, 2));
console.log(process.env.ENTERS_DIAGNOSTIC === '1' ? 'DIAGNOSTIC ONLY: coordinates require source verification' :
  `PASS: complete raw delta accounted; ${changes.length} exact changes, two deleted phase functions, all ordinary identities/values/budgets/totals fixed`);
