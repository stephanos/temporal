import fs from 'node:fs';
import path from 'node:path';
import {repo, out, base, git, hash} from './run.mjs';

const Replace = {};
for (const name of ['qualification_manifest.go', 'protocol.go', 'version.go', 'boundary.go']) {
  const relative = 'tools/gomad3/cmd/gomadtool/' + name;
  const bytes = git('show', base + ':' + relative);
  const destination = path.join(out, 'baseline-' + name);
  fs.writeFileSync(destination, bytes, {flag: 'wx'});
  Replace[path.join(repo, relative)] = destination;
  console.log(JSON.stringify({baseline: base, relative, destination, sha256: hash(bytes)}));
}
fs.writeFileSync(path.join(out, 'baseline-overlay.json'), JSON.stringify({Replace}, null, 2) + '\n', {flag: 'wx'});
