import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
const out=path.dirname(new URL(import.meta.url).pathname),config=JSON.parse(fs.readFileSync(path.join(out,'final-control-inputs.json'))),hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex'),files=[];
for(const name of ['base','current']){const receiptPath=path.join(out,'frozen-adapter-compile-'+name+'-receipt.json'),receipt=JSON.parse(fs.readFileSync(receiptPath));if(receipt.exit_code!==0||!receipt.source_unchanged||!receipt.controls_unchanged||!receipt.tools_unchanged)throw Error('adapter compile not complete and frozen');for(const file of [path.join(config.scratch,name+'-probe'),receiptPath,path.join(out,receipt.stdout.path),path.join(out,receipt.stderr.path)])files.push({path:file,sha256:hash(fs.readFileSync(file))});}
const metadata={files,cwd:config.adapter_cwd,roots:['base','current'].map(name=>path.join(config.scratch,'adapter-'+(name==='base'?'original':name)+'-root')),normalizer:config.normalizer,probe:config.probe,scope:'Real once-compiled ordinary probe executable inputs, frozen before running unchanged29cases from identical caller cwd and same-depth roots; no patched/native compiler qualification.'};
fs.writeFileSync(path.join(out,'adapter-executable-inputs.json'),JSON.stringify(metadata,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify(metadata));
