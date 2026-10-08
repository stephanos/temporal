import assert from 'node:assert/strict';
import {readdirSync,existsSync} from 'node:fs';
import {resolve} from 'node:path';
import {out,root,read,write,sha} from './capture.mjs';
const receipts=readdirSync(out).filter(n=>n.endsWith('.json')).flatMap(n=>{try{const r=JSON.parse(read(out+'/'+n));return r.label&&r.argv?[r]:[];}catch{return[];}});
const suites=receipts.filter(r=>r.argv.includes('test')&&r.argv.includes('-json')).map(r=>{
 const overlayIndex=r.argv.indexOf('-overlay'),overlay=overlayIndex>=0?JSON.parse(read(r.argv[overlayIndex+1])):null;
 const events=read(out+'/'+r.label+'.stdout').toString().split('\n').flatMap(l=>{try{return[JSON.parse(l)];}catch{return[];}});
 const assertions=events.filter(e=>e.Output&&(e.OutputType==='error'||e.OutputType==='error-continue')).map(e=>{
  const bindings=[...e.Output.matchAll(/([A-Za-z0-9_]+\.go):(\d+)/g)].map(m=>{const source=e.Package.replace('go.temporal.io/server/','')+'/'+m[1],absolute=resolve(root,source),actual=overlay?.Replace[absolute]??source;if(!existsSync(resolve(root,actual)))return {logical_path:source,line:Number(m[2]),source_absent:true};return {logical_path:source,actual_input_path:actual,line:Number(m[2]),sha256:sha(read(actual)),statement:read(actual).toString().split('\n')[Number(m[2])-1],counterfactual_overlay:Boolean(overlay?.Replace[absolute])};});
  return {test:e.Test??null,output_type:e.OutputType,raw:e.Output,bindings};
 });
 return {label:r.label,exit:r.exit,overlay:overlay?{path:r.argv[overlayIndex+1],sha256:sha(read(r.argv[overlayIndex+1])),replacement_inputs:Object.entries(overlay.Replace).map(([logical_path,path])=>({logical_path,path,sha256:sha(read(path))}))}:null,assertions,untruncated_stdout_sha256:r.stdout_sha256,untruncated_stderr_sha256:r.stderr_sha256};
});
assert(suites.find(s=>s.label==='sealed-causes-mutant').assertions.some(a=>a.output_type==='error-continue'&&a.raw.includes('invalid choice terminal values')));
write('terminal-failure-bindings.json',{suites,complete_raw_errors_and_continuations:true,overlay_sources_not_misattributed_to_current_file:true,cache_source_context:'Actual default stock GOCACHE overlay reports 0 available in sealed-disk-space receipt. Exhaustion is a likely cause of missing srcfiles entries, an inference; no cleanup, reset, copy or environment substitution on worker lane.',native:false,review_verdict:null});
console.log(JSON.stringify({suites:suites.length,raw_assertion_lines:suites.reduce((n,s)=>n+s.assertions.length,0)}));
