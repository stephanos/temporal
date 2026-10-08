import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

const out=path.dirname(new URL(import.meta.url).pathname);
const hash=bytes=>crypto.createHash('sha256').update(bytes).digest('hex');
export const sourceManifestIndex=()=>JSON.parse(fs.readFileSync(path.join(out,'source-manifest-index.json')));
export function sourceManifestBinding(original) {
 const row=sourceManifestIndex().bindings.find(row=>row.original===original);
 if(!row||path.basename(row.canonical)!==row.canonical)throw Error('unknown or unsafe source manifest '+original);
 const bytes=fs.readFileSync(path.join(out,row.canonical));
 if(bytes.length!==row.original_bytes||hash(bytes)!==row.original_payload_sha256||hash(JSON.stringify(JSON.parse(bytes)))!==row.source_tree_sha256)throw Error('source manifest binding mismatch '+original);
 const originalPath=path.join(out,original);
 if(fs.existsSync(originalPath)&&!fs.readFileSync(originalPath).equals(bytes))throw Error('original source manifest differs '+original);
 return {...row,index:'source-manifest-index.json'};
}
export function readSourceManifest(original) {
 const row=sourceManifestBinding(original);
 return JSON.parse(fs.readFileSync(path.join(out,row.canonical)));
}
