import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {read,out,sha,sources} from './capture.mjs';
import {functionSpans} from './proof.mjs';
const path='tools/gomad3/runner/completion_characterization_test.go',name='TestCompletionFaultsKeepReasonPrecedenceAndEvidence';
export function expectedCompletion(before){
 let expected=before.replace('\t\tsimulationCause string\n','\t\tsimulationCause string\n\t\tstatistics      *campaign.CampaignStatistics\n');
 expected=expected.replace('\t\t\tname: "World seed mismatch", fault: seedMismatch,\n','\t\t\tname: "World seed mismatch", fault: seedMismatch,\n\t\t\tstatistics:  &campaign.CampaignStatistics{Attempted: 1, DistinctFailures: 1},\n');
 expected=expected.replace('\t\t\tname: "malformed semantic coverage", coverage: CoverageSemantic, fault: malformedCoverage,\n','\t\t\tname: "malformed semantic coverage", coverage: CoverageSemantic, fault: malformedCoverage,\n\t\t\tstatistics:  &campaign.CampaignStatistics{Attempted: 1},\n');
 const existing='\t\t\t\tif observed := observeCompletion(t, summary, err); !reflect.DeepEqual(observed, want) {\n\t\t\t\t\tt.Fatalf("completion = %#v, want %#v", observed, want)\n\t\t\t\t}\n';
 assert.equal(expected.split(existing).length,2);
 return expected.replace(existing,existing+'\t\t\t\tif strategy == StrategySeed && test.statistics != nil {\n\t\t\t\t\tif observed := observeSeedCompletion(t, summary, err).Statistics; observed != *test.statistics {\n\t\t\t\t\t\tt.Fatalf("statistics = %#v, want %#v", observed, *test.statistics)\n\t\t\t\t\t}\n\t\t\t\t}\n');
}
export function completionDelta(){const before=read(out+'/completion-test-before.txt').toString(),after=read(path).toString();assert.equal(after,expectedCompletion(before));const fn=s=>functionSpans(s).find(f=>f.signature.startsWith('func '+name+'('));return {path,name,before_sha256:sha(before),after_sha256:sha(after),before_function:fn(before),after_function:fn(after),exact_additive_two_rows:true,assertion_execution:'unexecuted: same preparation validation guard before observeCompletion and before added Statistics assertions; native owners fn149/fn128'};}
if(process.argv[1]===fileURLToPath(import.meta.url)){if(process.argv[2]==='before'){writeFileSync(out+'/completion-test-before.txt',read(path),{flag:'wx'});writeFileSync(out+'/completion-before.json',JSON.stringify({path,sha256:sha(read(path)),source:sources()},null,2)+'\n',{flag:'wx'});}else{const p=completionDelta();writeFileSync(out+'/completion-additive-after.json',JSON.stringify(p,null,2)+'\n',{flag:'wx'});}console.log(JSON.stringify({mode:process.argv[2],path,sha256:sha(read(path))}));}
