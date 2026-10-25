import json, os, pathlib, sys, time
owner=int(sys.argv[1]); root=pathlib.Path.cwd().resolve(); prefix=str(root/'model/build/tlc')+'/'
started=time.monotonic(); seen={}; peak_model=peak_all=0; samples=0
while time.monotonic()-started<2800:
    try: os.kill(owner,0)
    except ProcessLookupError: break
    running=[]
    for proc in pathlib.Path('/proc').iterdir():
        if not proc.name.isdigit(): continue
        try:
            argv=(proc/'cmdline').read_bytes().split(b'\0')
            if not argv or pathlib.Path(os.fsdecode(argv[0])).name!='java': continue
            cwd=os.readlink(proc/'cwd')
            if not cwd.startswith(prefix): continue
            category='model' if b'tlc2.TLC' in argv and b'-tool' in argv else 'admission'
            pid=int(proc.name); running.append((pid,category)); seen[pid]={'category':category,'cwd':cwd}
        except (FileNotFoundError,ProcessLookupError,PermissionError): pass
    peak_model=max(peak_model,sum(kind=='model' for _,kind in running)); peak_all=max(peak_all,len(running)); samples+=1
    time.sleep(.05)
path=pathlib.Path(sys.argv[2]); path.write_text(json.dumps({'observation':'sampled live processes; not an exact launch inventory','sample_period_seconds':.05,'samples':samples,'seconds':round(time.monotonic()-started,3),'peak_model_jvms':peak_model,'peak_including_admission':peak_all,'observed_distinct_model_pids':sum(v['category']=='model' for v in seen.values()),'processes':seen},indent=2)+'\n')
print(path.read_text()[:320])
