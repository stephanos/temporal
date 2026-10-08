import assert from 'node:assert/strict';
import {readdirSync,existsSync} from 'node:fs';
import {resolve} from 'node:path';
import {read,write as writeOriginal,git,sha,out,root,sources} from './capture.mjs';
const freeze=process.argv.includes('--freeze');
const write=(n,v)=>writeOriginal(freeze?'final-'+n:n,n==='restoration-analysis.json'?{...v,completion:completionBinding}:v);
const fn=(s,n)=>{const start=s.indexOf('func '+n+'(');assert(start>=0,n);return s.slice(start,s.indexOf('\n}',start)+2)+'\n';};
const ref=path=>({path,sha256:sha(read(path)),bytes:read(path).length});
const arch='tools/gomad3/architecture_test.go',adapter='tools/gomad3/deterministicio/adapter_rewrite_test.go';
const beforeArch=read(out+'/before-architecture_test.go').toString(),nowArch=read(arch).toString(),oldAlias=fn(git(['show','97f221c7ea^:'+arch]).toString(),'TestPublicPackagesDoNotExportTypeAliases'),newAlias=fn(nowArch,'TestPublicPackagesDoNotExportForwardingAliases');
assert.equal(oldAlias.replace('TestPublicPackagesDoNotExportTypeAliases','TestPublicPackagesDoNotExportForwardingAliases'),newAlias);
assert.equal(nowArch,beforeArch+'\n'+newAlias);assert.equal(fn(nowArch,'TestPublicPackagesDoNotExportTypeAliases'),fn(beforeArch,'TestPublicPackagesDoNotExportTypeAliases'));
const oldAdapter=read(out+'/before-adapter_rewrite_test.go').toString(),nowAdapter=read(adapter).toString(),identity='TestRewrittenModulesRejectChangedIdentity',oldBody=fn(oldAdapter,identity),newBody=fn(nowAdapter,identity);
assert.equal(oldAdapter.replace(oldBody,''),nowAdapter.replace(newBody,''));
assert(newBody.includes('[]string{"empty", "populated"}')&&newBody.includes('cache == "populated" && adapter.outsideServerGraph'));
assert(newBody.includes('err == nil || !strings.Contains(err.Error(), "identity mismatch")'));
const inputs=JSON.parse(read(out+'/control-inputs.json'));
const completionPath='tools/gomad3/runner/completion_test.go',completionBefore=git(['show','HEAD:'+completionPath]).toString(),completionCurrent=read(completionPath).toString(),causeFunction='TestAssessCompletionProjectsCoverageInOrderAndClassifies';
assert.equal(sha(completionBefore),JSON.parse(read(out+'/completion-cause-control-inputs.json')).current_test.sha256);
const worldFunction='TestAssessWorldValidatesTheRecordAgainstItsSeed',worldRow='\t\t{name: "seed mismatch", record: recording, seed: 8, limit: 1 << 20, cause: "World record seed or schema does not match seed 8"},\n';
assert.equal(completionBefore.replace(fn(completionBefore,causeFunction),''),completionCurrent.replace(fn(completionCurrent,causeFunction),'').replace(worldRow,''));
assert.equal(fn(completionBefore,worldFunction),fn(completionCurrent,worldFunction).replace(worldRow,''));
const originalCause=fn(git(['show','59ca3d17395be5501906586006e2539d33bea28c:'+completionPath]).toString(),causeFunction),currentCause=fn(completionCurrent,causeFunction);
for(const line of originalCause.split('\n').filter(l=>l.includes('cause:')))assert(currentCause.includes(line));
assert(currentCause.includes('hostError.Err == nil || hostError.Err.Error() != test.cause'));
if(existsSync(resolve(out,'before-completion_test.go')))assert.equal(read(out+'/before-completion_test.go').toString(),completionBefore);else writeOriginal('before-completion_test.go',completionBefore);
const completionBinding={path:completionPath,before_sha256:sha(completionBefore),before_bytes_provenance:'HEAD blob exactly equal pre-restoration control-input current_test hash; reconstructed retention, not a new pre-edit timestamp',current_sha256:sha(completionCurrent),off_both_admitted_function_bytes_exact:true,all_four_original_cause_rows_exact:true,original_explicit_nil_guard_retained:true,world_only_additive_original_seed_row:true,world_row:worldRow};
const source=sources(),allProduction=Object.keys(source).filter(p=>p.startsWith('tools/gomad3/deterministicio/')&&p.endsWith('.go')&&!p.endsWith('_test.go'));
const resolveValue=(expr,revision)=>{
 if(expr.startsWith('"'))return JSON.parse(expr);
 for(const path of allProduction){let text;if(revision){if(!git(['ls-tree',revision,'--',path]).length)continue;text=git(['show',revision+':'+path]).toString();}else text=read(path).toString();const m=text.match(new RegExp('\\b'+expr+'\\s*=\\s*("[^"\\n]*")'));if(m)return JSON.parse(m[1]);}
 assert.fail('unresolved original/current pin '+expr);
};
const adapterBindings=inputs.original_functions.map(f=>{
 const start=nowAdapter.indexOf('prepare: prepare'+f.name.slice(4,-22));
 const identities=[...f.body.matchAll(/\{Module: ([^,]+), Version: ([^,]+), Sum: ([^}]+)\}/g)].map(m=>Object.fromEntries(['Module','Version','Sum'].map((k,i)=>[k,resolveValue(m[i+1].trim(),inputs.original_parent)])));
 assert.equal(identities.length,3);
 const prep=f.body.match(/err := (prepare\w+)\(/)[1],prepareIndex=nowAdapter.indexOf('prepare: '+prep+','),rowStart=nowAdapter.lastIndexOf('\n\t{',prepareIndex),rowEnd=nowAdapter.indexOf('\n\t},',prepareIndex)+5,row=nowAdapter.slice(rowStart,rowEnd);assert(prepareIndex>=0);
 const pins=row.match(/module: (\w+), version: (\w+), sum: (\w+)/);assert(pins);
 const Module=resolveValue(pins[1]),Version=resolveValue(pins[2]),Sum=resolveValue(pins[3]),otherModule=JSON.parse(row.match(/otherModule: ("[^"]+")/)[1]),otherVersion=JSON.parse(row.match(/otherVersion: ("[^"]+")/)[1]);
 const current=[{Module:otherModule,Version,Sum},{Module,Version:otherVersion,Sum},{Module,Version,Sum:'h1:changed'}];assert.deepEqual(current,identities);
 for(const pin of pins.slice(1))assert.equal(resolveValue(pin),resolveValue(pin,inputs.original_parent));
 return {original_function:f.name,original_body_sha256:f.sha256,current_template_row_sha256:sha(row),prepare:prep,original_identities:identities,current_identities:current,all_three_exact:true,empty_cache_replacement:identity+'/empty',populated_retained:identity+'/populated'};
});
write('restoration-analysis.json',{architecture:{before:ref(out+'/before-architecture_test.go'),current:ref(arch),exact_additive_renamed_original:true,current_accessibility_body_unchanged:true},adapter:{before:ref(out+'/before-adapter_rewrite_test.go'),current:ref(adapter),off_function_bytes_exact:true,empty_precedes_population:true,predicate_exact:true,adapterBindings},production_edits:git(['diff','--name-only','HEAD','--','tools/gomad3']).toString().trim().split('\n').filter(Boolean),review_verdict:null});
const events=label=>read(out+'/'+label+'.stdout').toString().split('\n').flatMap(l=>{try{return[JSON.parse(l)];}catch{return[];}}).filter(e=>e.Action);
const classify=lines=>lines.some(l=>/stat pinned Go command|Gomad .*driver|patched.*Go|\.toolchain\/bin\/go/.test(l))?'missing patched driver':lines.some(l=>/deterministic I\/O requires one of.*host is/.test(l))?'unsupported actual linux/arm64 host':lines.some(l=>/cache entry not found/.test(l))?'transient stock Go cache lookup':lines.some(l=>/test timed out/.test(l))?'suite timeout':lines.some(l=>/control changed|identity.*stat |exports forwarding alias/.test(l))?'expected meaningful negative control':'actual assertion failure; raw source/output retained, no invented native excuse';
const receipts=readdirSync(out).filter(p=>p.endsWith('.json')).flatMap(p=>{let v;try{v=JSON.parse(read(out+'/'+p));}catch{return[];}return v.label&&v.argv?[v]:[];});
const suites=receipts.filter(r=>r.tests.length||r.argv.includes('-json')&&r.argv.includes('test')).map(r=>{
 const ev=events(r.label),runs=ev.filter(e=>e.Action==='run'&&e.Test).map(e=>e.Test),finished=ev.filter(e=>['pass','fail','skip'].includes(e.Action)&&e.Test).map(e=>e.Test);
 const failures=r.tests.filter(t=>t.action==='fail').map(t=>{const lines=ev.filter(e=>e.Output&&(e.Test===t.test||e.Test?.startsWith(t.test+'/'))&&e.OutputType==='error').map(e=>e.Output);return {...t,classification:classify(lines),assertions:lines,assertion_sources:lines.flatMap(l=>[...l.matchAll(/([A-Za-z0-9_]+\.go):(\d+)/g)].map(m=>{const relative=t.package.replace('go.temporal.io/server/',''),p=relative+'/'+m[1];return existsSync(resolve(root,p))?{path:p,line:Number(m[2]),sha256:source[p],actual_line:read(p).toString().split('\n')[Number(m[2])-1]}:{path:p,line:Number(m[2]),source_absent_or_counterfactual:true};}))};});
 return {receipt:ref(out+'/'+r.label+'.json'),label:r.label,exit:r.exit,top_level:r.top_level_counts,subtests:r.subtest_counts,unfinished:runs.filter(n=>!finished.includes(n)),timeout:ev.filter(e=>e.Output?.includes('panic: test timed out')).map(e=>e.Output),failures,complete:!runs.some(n=>!finished.includes(n)),native:false};
});
const diagnostics=label=>{
 const lines=read(out+'/'+label+'.stdout').toString().split('\n');return lines.flatMap((text,i)=>{const m=text.match(/^(tools\/[^:]+):(\d+):(\d+): (.+) \(([^)]+)\)$/);if(!m)return[];const [_,path,line,column,message,linter]=m,content=read(path).toString(),statement=content.split('\n')[Number(line)-1],decls=[...content.slice(0,content.split('\n').slice(0,Number(line)).join('\n').length).matchAll(/^func ([^\n]+)/gm)],owner=git(['blame','--porcelain','-L',line+','+line,'--',path]).toString().split('\n')[0].split(' ')[0];const parent=owner.startsWith('0000')?null:git(['rev-parse',owner+'^']).toString().trim();return [{path,line:Number(line),column:Number(column),message,linter,raw:text,statement,function:decls.at(-1)?.[1]??'package scope',file_sha256:sha(read(path)),owner,parent,owner_patch_sha256:parent?sha(git(['diff',parent,owner,'--',path])):null,inherited_statement:git(['show','HEAD:'+path]).toString().split('\n').some(l=>l.trim()===statement.trim()),raw_statement:lines[i+1],waived:false}];});
};
const lint={scoped:diagnostics('final-unfiltered-scoped-lint'),fast:diagnostics('final-configured-fast-lint')};assert.equal(lint.scoped.length,92);assert.equal(lint.fast.length,3);
const strip=rows=>rows.map(({path,line,column,message,linter,statement})=>({path,line,column,message,linter,statement}));
assert.deepEqual(strip(lint.scoped).map(r=>({...r,line:undefined})),strip(diagnostics('unfiltered-scoped-lint')).map(r=>({...r,line:undefined})));
assert.deepEqual(strip(lint.fast),strip(diagnostics('configured-fast-lint')));
write('execution-analysis.json',{receipts:receipts.map(r=>({label:r.label,exit:r.exit,argv:r.argv,receipt:ref(out+'/'+r.label+'.json'),stdout:ref(out+'/'+r.label+'.stdout'),stderr:ref(out+'/'+r.label+'.stderr')})),suites,lint:{...lint,no_new_restoration_diagnostics:true,scoped_unfiltered:true,fast_selector_is_actual_task9_args:true,not_task6_fast_baseline:true},earlier_checker_controls:{label:'checker-controls',classification:'historical checker invocation omitted root label by using architecture; final-checker-controls corrects to all four actual labels, old observation not reused'},setup_collision:{label:'completion-cause-control-inputs',exit:1,classification:'outer capture basename EEXIST after child generated evidence inputs; original inputs and streams retained; no terminal receipt fabricated',stdout:ref(out+'/completion-cause-control-inputs.stdout'),stderr:ref(out+'/completion-cause-control-inputs.stderr')},native:false,full_host_pass:false,review_verdict:null});
const fixture='tools/gomad3/runner/testdata/diagnostic-identity-choices.json',originalFixture=JSON.parse(git(['show','f144997bd:'+fixture])),currentFixture=JSON.parse(read(fixture)),delta=[];
function diff(a,b,p=''){if(a===null||b===null||typeof a!=='object'||typeof b!=='object'){if(a!==b)delta.push({path:p,before:a,current:b});return;}assert.deepEqual(Object.keys(a).sort(),Object.keys(b).sort());for(const k of Object.keys(a))diff(a[k],b[k],p+'/'+k);}
diff(originalFixture,currentFixture);assert.equal(delta.length,7);
const derivation=JSON.parse(read(out+'/current-diagnostic-identity-derivation.stdout'));assert(derivation.current_golden_matches_candidate);assert.equal(derivation.candidate_fixture_sha256,'sha256:'+sha(read(fixture)));
write('diagnostic-fixture-binding.json',{fixture:ref(fixture),original_commit:git(['rev-parse','f144997bd']).toString().trim(),original_sha256:sha(git(['show','f144997bd:'+fixture])),delta,source_derivation:ref(out+'/current-diagnostic-identity-derivation.json'),source_only_helper:ref('.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-2/gfield-compact-20261005/identity-audit/derive.mjs'),derivation,classification:'Already v3 baseline; controller input identity and seven derived fields changed; not fn114 wire migration; unchanged trace payload and supplied input identities; no native execution'});
console.log(JSON.stringify({mapped_adapter_identity_sets:adapterBindings.length,suites:suites.length,scoped_lint:lint.scoped.length,fast_lint:lint.fast.length,fixture_delta:delta.length}));
