import assert from 'node:assert/strict';
import {resolve} from 'node:path';
import {spawnSync} from 'node:child_process';
import {out,root,stock,read,write,sha,environment} from './capture.mjs';
const production='tools/gomad3/runner/completion.go',source=read(production).toString(),anchor='\t\t\terr = fmt.Errorf("World record seed or schema does not match seed %d", seed)';
assert.equal(source.split(anchor).length,2);
const mutant=source.replace(anchor,anchor+'\n\t\t\tif seed == 8 && uint64(initialWorld.Config.Seed) == 7 {\n\t\t\t\terr = fmt.Errorf("control changed World mismatch cause")\n\t\t\t}');
write('control-world-seed-cause.go',mutant);
const prodReplace={[resolve(root,production)]:resolve(out,'control-world-seed-cause.go')},testReplace={[resolve(root,'tools/gomad3/runner/completion_test.go')]:resolve(out,'control-original-completion_test.go')};
write('control-world-current-mutant-overlay.json',{Replace:prodReplace});
write('control-world-original-mutant-overlay.json',{Replace:{...prodReplace,...testReplace}});
write('world-seed-input-binding.json',{production,production_sha256:sha(source),mutant_sha256:sha(mutant),condition:'Only World mismatch error text changes when expected seed 8 and decoded initial seed 7. Decode failures and seed7 expectations remain unchanged.',original_file_sha256:sha(read(out+'/control-original-completion_test.go')),current_file_sha256:sha(read('tools/gomad3/runner/completion_test.go')),native:false});
for(const [label,overlay] of [['world-original-seed-baseline','control-completion-original-normal-overlay.json'],['world-original-seed-mutant','control-world-original-mutant-overlay.json'],['world-current-seed-mutant','control-world-current-mutant-overlay.json']]){
 const r=spawnSync('node',[out+'/capture.mjs',label,stock+'/go','-C','tools/gomad3','test','-tags','test_dep','-count=1','-timeout=2m','-json','-overlay',out+'/'+overlay,'-run','^TestAssessWorldValidatesTheRecordAgainstItsSeed$','./runner'],{cwd:root,env:environment().env,stdio:'inherit',timeout:610000});assert(r.status!==null&&r.signal===null);
}
