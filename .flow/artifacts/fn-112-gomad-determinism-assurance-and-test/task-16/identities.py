import json,os,sys
c=sys.argv[1]
out=[]
for d in sorted(os.listdir(c+'/successes')):
    m=json.load(open(f'{c}/successes/{d}/manifest.json'))
    out.append({'dir':d,'seed':m['seed'],'record_hash':m['record_hash'],'failure_signature':m['outcome']['failure_signature'],'target_sha256':m['target']['sha256']})
print(json.dumps(sorted(out,key=lambda x:int(x['seed'])),indent=1))
