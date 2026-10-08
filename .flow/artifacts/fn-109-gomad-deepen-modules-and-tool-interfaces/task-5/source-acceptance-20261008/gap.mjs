import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {read,sha,git,board,sources,write} from './capture.mjs';
import {reconstruct,named,lineage} from './proof.mjs';
export function gap(){
 const path='tools/gomad3/cmd/gomad/internal/cli/cli.go',r=reconstruct(),original=named(r.images[path],'reportExploreFailure'),current=named(read(path).toString(),'reportExploreFailure');
 const merge='ca3345469a2541b7686606268b0f19cde030a8fe',local='a3b9f80efab9356c0be2080779133337e2471ac0',merged=named(git(['show',merge+':'+path]).toString(),'reportExploreFailure');
 assert(original.body.includes('errors.Join(writeErr, diagnosticWriteErr)'));assert(!merged.body.includes('errors.Join'));assert.equal(current.body,merged.body);
 const test='tools/gomad3/cmd/gomad/internal/cli/cli_test.go',guard=named(read(test).toString(),'TestExploreErrorReportsClassificationAfterChoiceDiagnosticWriterFailure');
 return {observed_at:new Date().toISOString(),kind:'actual task5 original postimage preservation gap',source_identity_sha256:sha(JSON.stringify(sources())),board:board(),path,original:{...original,sha256:sha(original.body)},current_before_restoration:{...current,sha256:sha(current.body)},actual_local_owner:local,actual_merge_owner:merge,actual_merge_parent_hunks:lineage(path).filter(e=>e.commit===merge),regression_existing:{path:test,...guard,sha256:sha(guard.body),limitation:'Reporter succeeds while the diagnostic writer fails; does not assert the simultaneous reporter/diagnostic failure fallback bytes.'},preservation_gap:'Final fallback stderr payload no longer joins the diagnostic writer error with the reporter error. Both attempts/status3 still occur, but original error bytes were lost.',approved_digest_decision_applies:false,generic_waiver:false,product_edits_at_observation:[],native:false,review_verdict:null};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){write('source-gaps.json',gap());console.log('pre-restoration preservation gap retained');}
