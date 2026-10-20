#!/bin/bash
set -eu
binary=$1
work=$2
mkdir -m 700 -p "$work"
"$binary" fixture-record --output "$work/one.json" --seed 7
"$binary" fixture-record --output "$work/two.json" --seed 7
cmp "$work/one.json" "$work/two.json"
for fault in timestamp path; do
    "$binary" fixture-record --output "$work/$fault-one.json" --seed 7 --fault "$fault"
    "$binary" fixture-record --output "$work/$fault-two.json" --seed 7 --fault "$fault"
    if cmp -s "$work/$fault-one.json" "$work/$fault-two.json"; then
        exit 1
    fi
done
python3 - "$work/one.json" <<'PY'
import hashlib,json,sys
b=open(sys.argv[1],'rb').read()
v=json.loads(b)
assert v['schema']=='gomad.execution-record/v2'
assert v['seed']==7 and isinstance(v['seed'],int)
assert b==json.dumps(v,sort_keys=True,separators=(',',':'),ensure_ascii=False).encode()
assert 'timestamp' not in v and 'path' not in v
print(hashlib.sha256(b).hexdigest())
PY
