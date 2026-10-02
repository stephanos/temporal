#!/usr/bin/env python3
"""Summarizes gomad qualification records (qualifications/v1/*.json) under an artifacts dir."""
import json, glob, sys, os, hashlib
root = sys.argv[1]
for p in sorted(glob.glob(root + '/qualifications/v1/*.json')):
    q = json.load(open(p))
    ev = q.get('evidence', {})
    print(f"seed={q.get('seed')} qualified={q.get('qualified')} deterministic={q.get('deterministic')} target_success={q.get('target_success')} first_divergence={q.get('first_divergence')} classification={q.get('classification')}")
    print(f"  evidence[0]: outcome={ev.get('outcome')} vt={ev.get('virtual_time_elapsed_nanos')} peak_g={ev.get('peak_goroutines')} stderr_bytes={ev.get('stderr',{}).get('total_bytes')} stdout_bytes={ev.get('stdout',{}).get('total_bytes')} io_records={ev.get('io_transcript_records')} terminal={ev.get('world',{}).get('terminal')} choice={ {k:v for k,v in ev.items() if 'choice' in k} }")
    for i, e in enumerate(q.get('executions', [])):
        ap = e.get('artifact_path', '')
        extra = ''
        if ap and os.path.isdir(ap):
            m = json.load(open(ap + '/manifest.json'))
            mev = m.get('evidence', m)
            extra = f" kind={os.path.basename(os.path.dirname(ap))} files={sorted(os.listdir(ap))}"
        print(f"  exec[{i}]: digest={e.get('evidence_digest','')[:20]} wall={int(e.get('wall_elapsed_nanos',0))/1e9:.1f}s replay={e.get('replay')} art={ap[len(root):]}{extra}")
    for k in q:
        if k not in ('command','evidence','executions','seed','qualified','deterministic','target_success','first_divergence','schema','repeat','evidence_digest','classification'):
            print(f"  {k}: {json.dumps(q[k])[:600]}")
