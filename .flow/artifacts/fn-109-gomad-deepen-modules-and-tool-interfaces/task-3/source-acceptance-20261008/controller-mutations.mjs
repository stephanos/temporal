import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {root,out,stock,read,sha} from './capture.mjs';
const path='tools/gomad3/runner/internal/campaign/controller.go',source=read(path).toString();
const needle='\tif completion.Kind == CompletionSuccess {',replacement='\tif completion.Kind == CompletionUnclassified || completion.Kind == CompletionSuccess {';
assert.equal(source.split(needle).length,2);
const mutant=source.replace(needle,replacement);
writeFileSync(out+'/controller-wrongly-classified-success.go.txt',mutant,{flag:'wx'});
writeFileSync(out+'/controller-wrongly-classified-success-overlay.json',JSON.stringify({Replace:{[root+'/'+path]:out+'/controller-wrongly-classified-success.go.txt'}},null,2)+'\n',{flag:'wx'});
writeFileSync(out+'/controller-mutation-binding.json',JSON.stringify({scope:'pure Controller behavior sensitivity only, not Runner publication assertion execution',source:{path,sha256:sha(source)},mutants:[{overlay:'missing-unclassified-attempted-overlay.json',sha256:sha(read(out+'/missing-unclassified-attempted.go.txt'))},{overlay:'controller-wrongly-classified-success-overlay.json',sha256:sha(mutant)}],selector:'^TestSeedControllerCompletionKeepsWholeStatistics$/^unclassified_attempt$'},null,2)+'\n',{flag:'wx'});
for(const [label,overlay] of [['controller-missing-attempted','missing-unclassified-attempted-overlay.json'],['controller-wrong-success','controller-wrongly-classified-success-overlay.json']]){
 const result=spawnSync('node',[out+'/capture.mjs',label,stock+'/go','-C','tools/gomad3','test','-json','-count=1','-tags','test_dep','-overlay='+out+'/'+overlay,'./runner/internal/campaign','-run','^TestSeedControllerCompletionKeepsWholeStatistics$/^unclassified_attempt$'],{cwd:root,stdio:'inherit',timeout:610000});assert.equal(result.status,1,'mutation must be rejected');
}
