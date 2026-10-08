import assert from 'node:assert/strict';
import {writeFileSync} from 'node:fs';
import {read,sha,out,sources,root} from './capture.mjs';
import {functionSpans} from './proof.mjs';
const path='tools/gomad3/runner/runner_test.go';
const name='TestRunRejectsSuccessfulRetentionWithoutReplayTranscript';
const source=read(path).toString();
const fn=functionSpans(source).find(f=>f.signature.startsWith('func '+name+'('));
assert(fn);
const mode=process.argv[2];
if(mode==='before') {
 writeFileSync(out+'/runner-test-before.txt',source,{flag:'wx'});
 writeFileSync(out+'/additive-before.json',JSON.stringify({path,name,sha256:sha(source),function:fn,source:sources()},null,2)+'\n',{flag:'wx'});
} else if(mode==='after') {
 const before=read(out+'/runner-test-before.txt').toString();
 const old=functionSpans(before).find(f=>f.signature.startsWith('func '+name+'('));
 const expected=old.body.replace('\t_, err := exploreWith','\tsummary, err := exploreWith').replace('\n}\n','\n\tif got := observeSeedCompletion(t, summary, err).Statistics; got != (campaign.CampaignStatistics{Attempted: 1}) {\n\t\tt.Fatalf("statistics = %#v, want %#v", got, campaign.CampaignStatistics{Attempted: 1})\n\t}\n}\n');
 assert.equal(fn.body,expected);
 assert.equal(source,before.replace(old.body,expected));
 const controller='tools/gomad3/runner/internal/campaign/controller.go';
 const c=read(controller).toString(),needle='\tcontroller.statistics.Attempted++';
 assert.equal(c.split(needle).length,2);
 const mutants=[{name:'missing-unclassified-attempted',path:controller,source:c.replace(needle,'\tif completion.Kind != CompletionUnclassified {\n\t\tcontroller.statistics.Attempted++\n\t}')},{name:'wrongly-classified-success',path:'tools/gomad3/runner/runner.go'}];
 const runner=read(mutants[1].path).toString(),retention='if retentionErr != nil {';
 const at=runner.indexOf(retention);assert(at>0);
 const tail=runner.slice(at),unclassified='recordCompletion(campaign.CompletedUnclassified())';
 const position=tail.indexOf(unclassified);assert(position>0&&position<2000);
 mutants[1].source=runner.slice(0,at)+tail.slice(0,position)+tail.slice(position).replace(unclassified,'recordCompletion(campaign.CompletedSuccess())');
 for(const m of mutants){writeFileSync(out+'/'+m.name+'.go.txt',m.source,{flag:'wx'});writeFileSync(out+'/'+m.name+'-overlay.json',JSON.stringify({Replace:{[root+'/'+m.path]:out+'/'+m.name+'.go.txt'}},null,2)+'\n',{flag:'wx'});}
 writeFileSync(out+'/additive-after.json',JSON.stringify({path,name,before_sha256:sha(before),after_sha256:sha(source),before_function:old,after_function:fn,exact_single_function_delta:true,original_statistics_vector:{Attempted:1,Succeeded:0,Failures:0,Watchdogs:0,ReplayDivergences:0,Cancelled:0,DistinctFailures:0,StopReason:''},mutants:mutants.map(m=>({name:m.name,path:m.path,sha256:sha(m.source),execution:'not attempted: canonical test refused before executor by unchanged unsupported native host guard; no meaningful mutation RED can be obtained here'}))},null,2)+'\n',{flag:'wx'});
} else throw new Error('before or after required');
console.log(JSON.stringify({mode,path,sha256:sha(source)}));
