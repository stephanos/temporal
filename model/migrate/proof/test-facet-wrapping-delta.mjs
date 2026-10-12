import fs from 'node:fs';
import assert from 'node:assert/strict';

const [irPath, waiversPath] = process.argv.slice(2);
let checkDelta;
try { ({checkDelta} = await import('./check-facet-wrapping-delta.mjs')); }
catch (error) {
  if (error.code !== 'ERR_MODULE_NOT_FOUND') throw error;
  assert.fail('the complete raw facet-wrapping delta checker is missing');
}
const ir = fs.readFileSync(irPath), waivers = fs.readFileSync(waiversPath);
checkDelta(ir, ir, waivers, waivers);
const accepted = JSON.parse(ir);
const mutations = {
  declaration(x) { x.actions.pop(); },
  identity(x) { x.functions[0].name += '.corrupted'; },
  order(x) { [x.functions[0], x.functions[1]] = [x.functions[1], x.functions[0]]; },
  position(x) { x.actions[0].position.line += 1; },
  predicate(x) {
    const predicate = x.functions.find(f =>
      f.name === 'activityProduct.property.activityProduct.terminalStatesAreFinal');
    assert.ok(predicate, 'the independent actual Property predicate must exist');
    predicate.body = {literal: {bool: false}};
  },
  queryLimit(x) { x.queries[0].limits = {steps: '999'}; },
  queryTotal(x) { x.queries[0].total = '999'; }
};
for (const [name, mutate] of Object.entries(mutations)) {
  const changed = structuredClone(accepted);
  mutate(changed);
  assert.throws(() => checkDelta(ir, Buffer.from(JSON.stringify(changed)), waivers, waivers),
    /raw IR differs/, name);
}
assert.throws(() => checkDelta(ir, Buffer.concat([ir, Buffer.from('\n')]), waivers, waivers),
  /raw IR differs/, 'no whitespace normalization');
const changedWaivers = structuredClone(JSON.parse(waivers));
changedWaivers.waivers[0].because = 'changed waiver reason';
assert.throws(() => checkDelta(ir, ir, waivers, Buffer.from(JSON.stringify(changedWaivers))),
  /raw waivers differ/);
console.log('PASS: complete raw equality; nine independent declaration/identity/order/position/predicate/Query/waiver/byte corruptions refused');
