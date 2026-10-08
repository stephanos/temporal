import fs from 'node:fs';
import path from 'node:path';
import {repo,out,go,env,hash,run,git} from './run.mjs';

const quote=s=>"'"+s.replaceAll("'","'\\''")+"'";
const resume=process.argv[2]==='resume';
const reviewResume=process.argv[2]==='human-review';
const work=resume||reviewResume?'/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/fn1134-source-walk-wvbsFc':fs.mkdtempSync(path.join(env.TMPDIR,'fn1134-source-walk-'));
const root=path.join(work,'gomad3'), baseline=path.join(work,'baseline'), candidate=path.join(work,'candidate');
if(!resume&&!reviewResume) {fs.mkdirSync(root); fs.mkdirSync(baseline); fs.mkdirSync(candidate);}
for(const file of resume||reviewResume?[]:git('ls-files','tools/gomad3').trim().split('\n')) {
  const relative=file.slice('tools/gomad3/'.length), destination=path.join(root,relative);
  fs.mkdirSync(path.dirname(destination),{recursive:true});
  fs.copyFileSync(path.join(repo,file),destination);
}
for(const name of resume||reviewResume?[]:['go.mod','go.sum','main.go']) {
  const source=path.join(repo,'tools/gomad3/deterministicio/testdata/sprig',name);
  if(!fs.existsSync(source)) continue;
  for(const dir of [baseline,candidate]) fs.copyFileSync(source,path.join(dir,name));
}
const sourceBinding=git('ls-files','tools/gomad3').trim().split('\n').map(p=>({path:p,sha256:hash(fs.readFileSync(p))}));
const setup=[{kind:'automated fixture materialization',path:work,source_files:sourceBinding.length,description:'Copy current tracked Gomad source and existing Sprig fixture. No production pins or user checkout edits; no qualification.'}];
const priorNames=reviewResume?['walk-build-tool','walk-candidate-edit','walk-candidate-edit-cwd-corrected','walk-candidate-download','walk-pin-impact','walk-adapter-dryrun']:resume?['walk-build-tool','walk-candidate-edit']:[];
const commands=priorNames.map(name=>{const r=JSON.parse(fs.readFileSync(path.join(out,name+'-receipt.json')));return {name,command:r.command,receipt:name+'-receipt.json',exit_code:r.exit_code,observation:name==='walk-candidate-edit'?'inconclusive: wrong effective cwd; root edit restored':undefined};});
const execute=(name,command,expected=0,options={})=> {const receipt=run(name,command,String(expected),options); commands.push({name,command,receipt:name+'-receipt.json',exit_code:receipt.exit_code}); return receipt;};
const state=()=>{
  const files=[];
  for(const entry of sourceBinding) {
    const file=path.join(root,entry.path.slice('tools/gomad3/'.length));
    files.push({path:entry.path,sha256:fs.existsSync(file)?hash(fs.readFileSync(file)):null});
  }
  return files;
};
const initial=state();
const binary=path.join(work,'gomadtool');
if(!resume&&!reviewResume) execute('walk-build-tool','go -C tools/gomad3 build -tags test_dep -o '+quote(binary)+' ./cmd/gomadtool');
if(!reviewResume) execute(resume?'walk-candidate-edit-cwd-corrected':'walk-candidate-edit',quote(go)+' mod edit -require=github.com/Masterminds/sprig/v3@v3.2.3',0,{cwd:candidate});
if(!reviewResume) execute('walk-candidate-download',quote(go)+' mod download github.com/Masterminds/sprig/v3@v3.2.3',0,{cwd:candidate});
const candidateBefore=['go.mod','go.sum'].map(p=>({path:p,sha256:hash(fs.readFileSync(path.join(candidate,p)))}));
if(!reviewResume) execute('walk-pin-impact',quote(binary)+' pin-impact --root='+quote(root)+' --baseline='+quote(path.join(baseline,'go.mod'))+' --candidate='+quote(path.join(candidate,'go.mod'))+' --go='+quote(go)+' --format=json',1);
const impact=JSON.parse(fs.readFileSync(path.join(out,'walk-pin-impact.log'),'utf8'));
const sprigPin=impact.pins.find(p=>p.class==='adapter' && p.module==='github.com/Masterminds/sprig/v3');
if(!sprigPin || sprigPin.status!=='invalidated') throw Error('impact did not invalidate exact Sprig pin');
if(!reviewResume) execute('walk-adapter-dryrun',quote(binary)+' adapter-regenerate --root='+quote(root)+' --module=github.com/Masterminds/sprig/v3 --version=v3.2.3 --go='+quote(go)+' --json');
const review=JSON.parse(fs.readFileSync(path.join(out,'walk-adapter-dryrun.log'),'utf8'));
if(JSON.stringify(state())!==JSON.stringify(initial) || review.applied) throw Error('dry run changed source');
execute('walk-adapter-human-review',quote(binary)+' adapter-regenerate --root='+quote(root)+' --module=github.com/Masterminds/sprig/v3 --version=v3.2.3 --go='+quote(go));
const human=fs.readFileSync(path.join(out,'walk-adapter-human-review.log'),'utf8');
const approval=human.match(/^approval: (sha256:[a-f0-9]{64})$/m)?.[1];
if(!approval || JSON.stringify(state())!==JSON.stringify(initial)) throw Error('human dry run invalid');
fs.writeFileSync(path.join(out,'walk-setup.json'),JSON.stringify({work,root,baseline,candidate,binary,binary_sha256:hash(fs.readFileSync(binary)),source_binding:sourceBinding,initial_scratch:initial,setup,commands,review_digest:approval,candidate_before:candidateBefore,additional_review_invocation:'JSON preserves its existing contract without an approval field; extra human-rendered dry run is an actual counted invocation, not a CLI change.'},null,2)+'\n');
console.log('Review required before approved apply: '+approval);
