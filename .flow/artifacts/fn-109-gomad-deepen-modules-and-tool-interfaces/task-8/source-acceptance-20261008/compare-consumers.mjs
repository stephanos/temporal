import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const [original='matched-original-counted',current='matched-current-counted',prefix='consumer-comparison',expectedText='27']=process.argv.slice(2);
const expected=Number(expectedText);
const key=row=>JSON.stringify([row.test,row.name,row.operation,row.arguments,row.failed_writer,row.failed_removal,row.invalid_review]);
function observations(name){
 const receipt=JSON.parse(fs.readFileSync(path.join(out,name+'-receipt.json'))),bytes=fs.readFileSync(path.join(out,receipt.log));
 if(hash(bytes)!==receipt.log_sha256||receipt.exit_code!==0||!receipt.source_unchanged||!receipt.controls_unchanged||!receipt.tools_unchanged)throw Error('unqualified comparison receipt '+name);
 const groups=new Map();for(const line of bytes.toString().split('\n')){let event;try{event=JSON.parse(line)}catch{continue}if(event.Action==='output'&&event.Test){const key=event.Package+'|'+event.Test;groups.set(key,(groups.get(key)??'')+event.Output)}}
 const rows=[];for(const [test,text]of groups)for(const match of text.matchAll(/SOURCE_OBSERVATION (.+)\n/g))rows.push({test,...JSON.parse(match[1])});
 if(rows.length!==expected)throw Error('expected '+expected+' complete observations, saw '+rows.length+' '+name);
 if(new Set(rows.map(key)).size!==rows.length)throw Error('duplicate observation identity '+name);
 rows.sort((a,b)=>key(a).localeCompare(key(b)));
 const snapshot=JSON.stringify(rows,null,2)+'\n';fs.writeFileSync(path.join(out,name+'-observations.json'),snapshot);
 return {receipt,snapshot_sha256:hash(snapshot),rows};
}
const old=observations(original),now=observations(current);
const differences=[];for(let i=0;i<old.rows.length;i++){
 const left=old.rows[i],right=now.rows[i];if(key(left)!==key(right))throw Error('observation identity differs '+i);
 for(const field of new Set([...Object.keys(left),...Object.keys(right)]))if(JSON.stringify(left[field])!==JSON.stringify(right[field]))differences.push({test:left.test,field,original:left[field],current:right[field]});
}
const result={original,current,original_log_sha256:old.receipt.log_sha256,current_log_sha256:now.receipt.log_sha256,original_snapshot_sha256:old.snapshot_sha256,current_snapshot_sha256:now.snapshot_sha256,rows:old.rows.length,differences,source_native:false};
fs.writeFileSync(path.join(out,prefix+'.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({rows:result.rows,differences:differences.map(d=>({test:d.test,field:d.field,original:d.field==='status'?d.original:undefined,current:d.field==='status'?d.current:undefined}))}));
