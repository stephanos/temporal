from pathlib import Path
import subprocess, os, sys, json, datetime, hashlib
root = Path(__file__).resolve().parent
name = sys.argv[1]
command = sys.argv[2:]
env = dict(os.environ)
for key in ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED']:
    env.pop(key, None)
env['GOFLAGS'] = '-tags=test_dep'
env['GOMAD3_STOCK_GO'] = '/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/go'
env['PATH'] = str(Path(env['GOMAD3_STOCK_GO']).parent) + ':' + env['PATH']
env['GOLANGCI_LINT_BASE_REV'] = 'HEAD'
record = {'command': command, 'cwd': os.getcwd(), 'unset': ['GOROOT','GOMADSEED','GOMAD3_CHILD_SEED'], 'environment': {key: env[key] for key in ['GOFLAGS','GOMAD3_STOCK_GO','PATH','GOLANGCI_LINT_BASE_REV']}, 'started': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'source_manifest_sha256': hashlib.sha256((root/'source-bindings.json').read_bytes()).hexdigest()}
with (root/(name+'.log')).open('wb') as output:
    record['exit'] = subprocess.run(command, env=env, stdout=output, stderr=subprocess.STDOUT).returncode
record['finished'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
record['log_sha256'] = hashlib.sha256((root/(name+'.log')).read_bytes()).hexdigest()
(root/(name+'.json')).write_text(json.dumps(record, indent=2)+'\n')
print(json.dumps({'gate': name, 'exit': record['exit']}))
sys.exit(record['exit'])
