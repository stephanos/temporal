import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),hash=b=>crypto.createHash('sha256').update(b).digest('hex');
const prefix=process.argv[2]??'';
function rows(name){
 const events=fs.readFileSync(path.join(out,prefix+'listing-boundary-'+name+'.stdout'),'utf8').trim().split('\n').map(JSON.parse),outputs=new Map();
 for(const event of events)if(event.Action==='output'&&event.Test)outputs.set(event.Test,(outputs.get(event.Test)??'')+event.Output);
 return [...outputs.values()].flatMap(value=>value.split('\n').filter(line=>line.startsWith('TASK9_ROW ')).map(line=>JSON.parse(line.slice(10))));
}
const original=rows('original'),historical=rows('historical-final'),current=rows('current');
const diff=other=>original.flatMap((row,index)=>JSON.stringify(row)===JSON.stringify(other[index])?[]:[{name:row.Name,original:row,current:other[index]}]);
const controls=JSON.parse(fs.readFileSync(path.join(out,prefix?'listing-control-v2-inputs.json':'listing-control-inputs.json'))),commands=fs.readFileSync(path.join(controls.scratch,'commands.jsonl'),'utf8').trim().split('\n').map(JSON.parse);
const reaping=commands.map(command=>{let gone=false;try{process.kill(command.pid,0)}catch(error){if(error.code!=='ESRCH')throw error;gone=true;}return {pid:command.pid,mode:command.mode,standard:command.standard,cwd:command.cwd,gone};});
const result={cases:original.length,normalization:'None; shared literal caller/control paths are identical across graphs.',original,historical,current,historical_differences:diff(historical),current_differences:diff(current),command_observations:{path:path.join(controls.scratch,'commands.jsonl'),sha256:hash(fs.readFileSync(path.join(controls.scratch,'commands.jsonl'))),count:commands.length,reaping},scope:'Public ReviewCapabilities and private standard-inventory entry. No fabricated provenance, target launch or compiler/native execution. Relevant default bounded/watchdog projection controls are separately retained.'};
fs.writeFileSync(path.join(out,prefix+'listing-boundary-comparison.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({cases:result.cases,historical_differences:result.historical_differences.map(x=>x.name),current_differences:result.current_differences.map(x=>x.name),children:commands.length,all_gone:reaping.every(x=>x.gone)}));
