import {readFileSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
const out=dirname(fileURLToPath(import.meta.url)),worker=resolve(out,'../source-acceptance-20261008');
const load=name=>JSON.parse(readFileSync(resolve(worker,name+'.json')));
function capture(label,argv,expected=0){const r=spawnSync('node',[resolve(out,'capture.mjs'),label,...argv],{encoding:'utf8',maxBuffer:64<<20});process.stdout.write(r.stdout??'');process.stderr.write(r.stderr??'');assert.equal(r.status,expected,label);}
capture('worker-integrity',['node',resolve(worker,'verify.mjs')]);
const full=load('portable-cli-full-temp-retry'),passed=full.tests.filter(t=>t.action==='pass'&&!t.test.includes('/'));
assert.equal(passed.length,89);assert.equal(full.tests.filter(t=>t.action==='fail'&&!t.test.includes('/')).length,3);
const go=full.argv[0],selector='^('+passed.map(t=>t.test).sort().join('|')+')$';
capture('portable-cli-passing-set',[go,'-C','tools/gomad3','test','-count=1','-tags','test_dep','-json','./cmd/gomad/internal/cli','-run',selector,'-timeout=90s']);
for(const label of ['portable-architecture-installation','portable-qualification-control','configured-generators','configured-cli-errortype','cli-format-check'])capture(label,load(label).argv);
for(const label of ['configured-fast-lint','unfiltered-cli-lint'])capture(label,load(label).argv,2);
