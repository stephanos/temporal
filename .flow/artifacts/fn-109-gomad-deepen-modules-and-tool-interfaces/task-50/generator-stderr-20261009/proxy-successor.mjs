import path from 'node:path';
import {run, proxyRoot, out} from './run.mjs';

run('proxy-archive-proof', 'node .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-50/generator-stderr-20261009/proxy-archive-proof.mjs', 0);
const environment = {GOPROXY: 'file://' + proxyRoot};
run('adapter-controls-offline-proxy', "go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool -run '^TestRunAdapterRegenerateOutputFailuresPreserveStatus$'", 0, environment);
run('ordinary-package-final-offline-proxy', 'go -C tools/gomad3 test -json -count=1 -tags test_dep ./cmd/gomadtool', 1, environment);
run('ordinary-package-baseline-offline-proxy', 'go -C tools/gomad3 test -json -count=1 -tags test_dep -overlay=' + path.join(out, 'baseline-overlay.json') + ' ./cmd/gomadtool', 1, environment);
console.log(JSON.stringify({source: 'unchanged frozen candidate', environment_change: environment, execution_lane: 'offline proxy successor children terminal'}));
