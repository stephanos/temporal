import hashlib, json, os, pathlib, subprocess, sys, time
root = pathlib.Path(__file__).resolve().parents[4]
module = root / 'tools/gomad3'
evidence = pathlib.Path(__file__).resolve().parent
stock = '/Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin'
env = os.environ.copy()
for key in ('GOROOT', 'GOBIN', 'GOMADSEED', 'GOMAD3_CHILD_SEED'):
    env.pop(key, None)
env.update(PATH=stock+os.pathsep+env['PATH'], GOMAD3_STOCK_GO=stock+'/go', GOEXPERIMENT='nogreenteagc', GOTOOLCHAIN='local', GOWORK='off')
name = sys.argv[1]
argv = sys.argv[2:]
cwd = root if name == 'full-host' else module
if name == 'full-host':
    env['GOFLAGS'] = '-tags=test_dep -count=1'
start = time.monotonic()
p = subprocess.run(argv, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
record = {'argv': argv, 'cwd': str(cwd), 'env': {k: env[k] for k in ('PATH','GOMAD3_STOCK_GO','GOEXPERIMENT','GOTOOLCHAIN','GOWORK') if k in env}, 'unset': ['GOROOT','GOBIN','GOMADSEED','GOMAD3_CHILD_SEED'], 'exit': p.returncode, 'duration_seconds': time.monotonic()-start, 'stdout_sha256': hashlib.sha256(p.stdout).hexdigest(), 'stderr_sha256': hashlib.sha256(p.stderr).hexdigest()}
if 'GOFLAGS' in env:
    record['env']['GOFLAGS'] = env['GOFLAGS']
(evidence / (name+'.stdout')).write_bytes(p.stdout)
(evidence / (name+'.stderr')).write_bytes(p.stderr)
(evidence / (name+'.json')).write_text(json.dumps(record,indent=2)+'\n')
print(name, 'exit', p.returncode, 'duration', round(record['duration_seconds'],2), flush=True)
