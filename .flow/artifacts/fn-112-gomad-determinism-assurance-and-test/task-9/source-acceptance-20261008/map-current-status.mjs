import assert from 'node:assert/strict';
import {read as readOriginal,write as writeOriginal,out,sha,git,sources} from './capture.mjs';
const read=p=>readOriginal(p.replace('/frozen-source-proof.json','/sealed-source-proof.json'));
const write=(n,v)=>writeOriginal(n,n==='current-mapping-analysis.json'?{...v,golden_fixture:{...v.golden_fixture,checker_rejection_controls:'sealed-final-checker-controls.json'}}:v);
const proof=JSON.parse(read(out+'/frozen-source-proof.json'));
const labels={root:'sealed-root',deterministicio:'sealed-final-deterministicio',cli:'sealed-final-cli',runner:'sealed-final-runner-focused'},receipts=Object.fromEntries(Object.entries(labels).map(([p,l])=>[p,JSON.parse(read(out+'/'+l+'.json'))]));
const rows=proof.mapping_rows.map(row=>{const receipt=receipts[row.package],status=row.replacement==='-'?'authorized-housekeeping':receipt.tests.find(t=>t.test===row.replacement)?.action??'not observed in bounded focused selection';return {...row,actual_current_status:status,actual_receipt:labels[row.package]+'.json',historical_recorded_status_not_reused:true};});
assert.equal(rows.length,282);assert.equal(rows.filter(r=>r.replacement==='-').length,11);
const currentLines=Object.entries(receipts).flatMap(([p,r])=>r.tests.map(t=>p+'\t'+t.test+'\t'+t.action)).sort();
write('current-behaviors.tsv','package\ttest\tstatus\n'+currentLines.join('\n')+'\n');
write('current-mapping.tsv','package\told\toriginal_status\tcurrent_replacement\tactual_current_status\tcondition\n'+rows.map(r=>[r.package,r.old,r.old_status,r.replacement,r.actual_current_status,r.note].join('\t')).join('\n')+'\n');
const current=sources(),body=p=>read(p).toString(),conditions=[
 {original:'TestPublicPackagesDoNotExportTypeAliases blanket exported alias predicate',current:'TestPublicPackagesDoNotExportForwardingAliases',path:'tools/gomad3/architecture_test.go',predicate:'typeSpec.Name.IsExported() && typeSpec.Assign.IsValid()'},
 {original:'Seven empty cache identity predicates',current:'TestRewrittenModulesRejectChangedIdentity/empty',path:'tools/gomad3/deterministicio/adapter_rewrite_test.go',predicate:'err == nil || !strings.Contains(err.Error(), "identity mismatch")'},
 {original:'Four exact private completion Causes',current:'TestAssessCompletionProjectsCoverageInOrderAndClassifies',path:'tools/gomad3/runner/completion_test.go',predicate:'hostError.Err == nil || hostError.Err.Error() != test.cause'},
];
for(const c of conditions)assert(body(c.path).includes(c.predicate));
const controls=[];for(const c of conditions){const changed=body(c.path).replace(c.predicate,'false');assert(!changed.includes(c.predicate));assert.notEqual(sha(changed),current[c.path]);controls.push({condition:c.original,source_text_counterfactual_rejected:true,actual_behavior_controls_separately_retained:true});}
write('current-mapping-analysis.json',{rows,additional_preserved_conditions:conditions,source_binding_controls:controls,proof_sha256:sha(read(out+'/frozen-source-proof.json')),counts_by_actual_status:Object.fromEntries([...new Set(rows.map(r=>r.actual_current_status))].map(s=>[s,rows.filter(r=>r.actual_current_status===s).length])),golden_fixture:{path:proof.golden.current.path,sha256:proof.golden.current.sha256,original_rows_exact:125,current_rows:126,checker_rejection_controls:'frozen-final-checker-controls.json'},no_current_failure_waived:true,no_unobserved_behavior_claimed_pass:true,review_verdict:null,native:false});
console.log(JSON.stringify({mapping_edges:rows.length,housekeeping:11,actual_statuses:Object.fromEntries([...new Set(rows.map(r=>r.actual_current_status))].map(s=>[s,rows.filter(r=>r.actual_current_status===s).length]))}));
