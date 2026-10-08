import {read,write,sha,git} from './capture.mjs';
const paths=['tools/gomad3/architecture_test.go','tools/gomad3/deterministicio/adapter_rewrite_test.go'];
for(const path of paths)write('before-'+path.split('/').at(-1),read(path));
write('before-restoration.json',{head:git(['rev-parse','HEAD']).toString().trim(),files:paths.map(path=>({path,sha256:sha(read(path))})),admission:{path:'.flow/tasks/fn-112-gomad-determinism-assurance-and-test.9.md',sha256:sha(read('.flow/tasks/fn-112-gomad-determinism-assurance-and-test.9.md'))}});
