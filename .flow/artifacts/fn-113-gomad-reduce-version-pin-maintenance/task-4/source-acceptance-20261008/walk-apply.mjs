import fs from 'node:fs';
import path from 'node:path';
import {repo,out,go,hash,run,git} from './run.mjs';
const quote=s=>"'"+s.replaceAll("'","'\\''")+"'";
const setup=JSON.parse(fs.readFileSync(path.join(out,'walk-setup.json')));
const root=path.join(setup.work,'temporal/tools/gomad3');
fs.mkdirSync(path.dirname(root),{recursive:true});
if(fs.existsSync(setup.root)) fs.renameSync(setup.root,root);
const parent=path.join(setup.work,'temporal');
const materialized=[];
for(const file of git('ls-files').trim().split('\n')) {
  if(file.startsWith('tools/gomad3/') || file.startsWith('.flow/') || file.startsWith('.turbo/')) continue;
  const source=path.join(repo,file),destination=path.join(parent,file);
  if(!fs.existsSync(source) || !fs.statSync(source).isFile()) continue;
  fs.mkdirSync(path.dirname(destination),{recursive:true});
  fs.copyFileSync(source,destination);
  materialized.push({path:file,sha256:hash(fs.readFileSync(source))});
}
const before=new Map(setup.initial_scratch.map(x=>[x.path,x.sha256]));
for(const [file,digest] of before) {
  if(hash(fs.readFileSync(path.join(root,file.slice('tools/gomad3/'.length))))!==digest) throw Error('scratch source changed before approval');
}
const review=JSON.parse(fs.readFileSync(path.join(out,'walk-adapter-dryrun.log')));
if(review.diffs.length!==1 || review.diffs[0].changed || review.diffs[0].path!=='network.go' || review.regeneration.previous.version!=='v3.3.0' || review.regeneration.proposed.version!=='v3.2.3' || review.stale_packs.length!==0) throw Error('review does not match inspected source');
const approval=setup.review_digest;
const inspected={approval,review_log:'walk-adapter-human-review.log',review_log_sha256:hash(fs.readFileSync(path.join(out,'walk-adapter-human-review.log'))),decision:'Controlled scratch source authoring only: same reviewed network.go bytes, exact version/sum and both prepared source sets change; no stale pack binding; no production approval or qualification.',native_qualification:false};
fs.writeFileSync(path.join(out,'walk-approval.json'),JSON.stringify(inspected,null,2)+'\n');
const receipt=run('walk-adapter-apply',quote(setup.binary)+' adapter-regenerate --root='+quote(root)+' --module=github.com/Masterminds/sprig/v3 --version=v3.2.3 --go='+quote(go)+' --approve-review='+quote(approval)+' --json');
const applied=JSON.parse(fs.readFileSync(path.join(out,'walk-adapter-apply.log')));
if(!applied.applied || applied.warnings?.length || !applied.published?.length) throw Error('publication did not complete cleanly');
const published=new Set(applied.published);
const postimage=setup.initial_scratch.map(entry=>{
  const relative=entry.path.slice('tools/gomad3/'.length);
  const sha256=hash(fs.readFileSync(path.join(root,relative)));
  if(sha256!==entry.sha256 && !published.has(relative)) throw Error('unlisted publication '+relative);
  return {path:relative,before:entry.sha256,after:sha256,published:published.has(relative)};
});
const outputBindings=applied.published.map(p=>({path:p,sha256:hash(fs.readFileSync(path.join(root,p)))}));
for(const required of ['deterministicio/sprig_adapter.go','toolchain/version/version.json','toolchain/version/generated.go','deterministicio/testdata/sprig/go.mod','deterministicio/testdata/sprig/go.sum','deterministicio/boundary/upgrade-go1.27.1.md']) if(!published.has(required)) throw Error('missing publication '+required);
const commands=[...setup.commands,{name:'walk-adapter-apply',command:receipt.command,receipt:'walk-adapter-apply-receipt.json',exit_code:receipt.exit_code}];
fs.writeFileSync(path.join(out,'walk-publication.json'),JSON.stringify({root,parent,materialized_source_files:materialized.length,materialized_source_sha256:hash(JSON.stringify(materialized)),approval:inspected,outputs:outputBindings,unchanged_source_paths:postimage.filter(p=>p.before===p.after).length,changed:postimage.filter(p=>p.before!==p.after),commands},null,2)+'\n');
const validation=run('walk-validate','make -C '+quote(root)+' validate');
commands.push({name:'walk-validate',command:validation.command,receipt:'walk-validate-receipt.json',exit_code:validation.exit_code});
fs.writeFileSync(path.join(out,'walkthrough.json'),JSON.stringify({schema:'gomad3.source-pin-maintenance-walk/v1',host:'linux/arm64 stock Go; not a qualified Gomad host',root,parent,binary_sha256:setup.binary_sha256,original_baseline:'../../task-1/baseline.json',matched_first_baseline:'../measurement.json',source_snapshot:JSON.parse(fs.readFileSync(path.join(out,'walk-build-tool-receipt.json'))).source_manifest,approval:inspected,outputs:outputBindings,commands,observed_command_invocations:commands.length,observed_hand_edits:1,recovery_hand_edit:'Restore only worker-caused root go.mod line after wrong-cwd attempt; root go.mod/go.sum restored byte-exact.',source_authoring_invocations:5,successful_common_preparation_invocations:2,tool_build_invocations:1,misplaced_inconclusive_invocations:1,normalized_source_authoring_invocations:4,matched_source_baseline_units:6,matched_source_baseline_with_common_preparation_units:8,actual_total_units:commands.length+1,native_stages:'Consumer test, patched runner rebuild, pack qualification, core qualification and full native test-host deferred to fn149/fn128; none executed or counted as source proof.',pack_reduction:'none measured; historical two-request projection stays historical',candidate_before:setup.candidate_before,candidate_after:setup.candidate_before.map(e=>({path:e.path,sha256:hash(fs.readFileSync(path.join(setup.candidate,e.path)))}))},null,2)+'\n');
console.log('walk terminal: '+commands.length+' command invocations + 1 recovery hand edit');
