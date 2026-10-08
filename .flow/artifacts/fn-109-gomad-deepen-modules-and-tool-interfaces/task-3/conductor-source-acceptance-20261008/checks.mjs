import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {root,read} from './capture.mjs';
const here=dirname(fileURLToPath(import.meta.url));
const worker='.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-3/source-acceptance-20261008';
const checks=[['campaign-state-machines',0],['final-portable-runner-boundaries',0],['portable-package-architecture',0],['controller-missing-attempted',1],['controller-wrong-success',1],['canonical-controller-after-mutations',0],['final-errortype',0],['final-generated-validation',0],['final-gofmt',0],['final-diff-check',0],['final-configured-fast-lint',2],['final-relevant-unfiltered-lint',2]];
for(const [name,expected] of checks){
 const receipt=JSON.parse(read(worker+'/'+name+'.json'));
 const result=spawnSync('node',[here+'/capture.mjs','conductor-'+name,...receipt.argv],{cwd:root,stdio:'inherit',timeout:610000});
 assert.equal(result.error,undefined,name+' transport');
 assert.equal(result.status,expected,name+' actual exit');
}
