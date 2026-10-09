import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const repo='/Users/stephan/Workspace/skunkworks/gomad/temporal',out=path.dirname(new URL(import.meta.url).pathname),base='29c80199cd';
const hash=b=>crypto.createHash('sha256').update(b).digest('hex');
function original(file){const r=spawnSync('git',['show',base+':'+file],{cwd:repo});if(r.status!==0)throw Error(r.stderr);return r.stdout.toString();}
function functions(source){
 const result={};
 for(const match of source.matchAll(/^func (?:(\([^\n]*?\)) )?(\w+)\(/gm)){
  let i=source.indexOf('{',match.index),depth=0,quote='',line=false,block=false,end=0;
  for(;i<source.length;i++){
   const c=source[i],next=source[i+1];
   if(line){if(c==='\n')line=false;continue;}if(block){if(c==='*'&&next==='/'){block=false;i++;}continue;}
   if(quote){if(c==='\\'&&quote!=='`'){i++;continue;}if(c===quote)quote='';continue;}
   if(c==='/'&&next==='/'){line=true;i++;continue;}if(c==='/'&&next==='*'){block=true;i++;continue;}
   if(['"',"'",'`'].includes(c)){quote=c;continue;}
   if(c==='{')depth++;if(c==='}'&&--depth===0){end=i+1;break;}
  }
  const receiver=match[1]?match[1].slice(1,-1).trim().split(/\s+/).at(-1).replace('*','')+'.':'';
  result[receiver+match[2]]=source.slice(match.index,end);
 }
 return result;
}
const command='tools/gomad3/target/internal/gocommand/command.go',old=functions(original(command)),current=functions(fs.readFileSync(path.join(repo,command),'utf8'));
const methods=Object.keys(old).map(name=>({name,before_sha256:hash(old[name]),after_sha256:hash(current[name]),equal:old[name]===current[name]}));
const unix='tools/gomad3/internal/hostexec/command_unix.go',request='tools/gomad3/internal/hostexec/command.go';
const inverseUnix=fs.readFileSync(path.join(repo,unix),'utf8').replace('\tif request.CombinedOutput {\n\t\tcommand.Stderr = stdoutWrite\n\t}\n','');
const inverseRequest=fs.readFileSync(path.join(repo,request),'utf8').replace(/^\tCombinedOutput\s+bool\n/m,'');
const ls=spawnSync('git',['ls-files','tools/gomad3'],{cwd:repo,encoding:'utf8'});if(ls.status!==0)throw Error(ls.stderr);
const tests=ls.stdout.trim().split('\n').filter(p=>p.endsWith('_test.go')).map(file=>({path:file,before_sha256:hash(original(file)),after_sha256:hash(fs.readFileSync(path.join(repo,file))),equal:original(file)===fs.readFileSync(path.join(repo,file),'utf8')}));
const result={base,methods,hostexec_inverse_exact:inverseUnix===original(unix),request_inverse_exact:inverseRequest===original(request),existing_tests:tests,new_methods:Object.keys(current).filter(x=>!Object.hasOwn(old,x)),limitations:'Exact source preservation of existing operations and test files; not a runtime review verdict or genuine deferred-close fault execution.'};
fs.writeFileSync(path.join(out,'default-preservation.json'),JSON.stringify(result,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({default_methods:methods.length,unchanged:methods.every(x=>x.equal),hostexec_inverse_exact:result.hostexec_inverse_exact,request_inverse_exact:result.request_inverse_exact,existing_tests:tests.length,existing_tests_unchanged:tests.every(x=>x.equal),new_methods:result.new_methods}));
if(methods.some(x=>!x.equal)||!result.hostexec_inverse_exact||!result.request_inverse_exact||tests.some(x=>!x.equal))process.exitCode=1;
