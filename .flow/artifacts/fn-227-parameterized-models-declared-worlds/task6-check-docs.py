from pathlib import Path
import re
for file in ['model/DESIGN.md','model/README.md']:
 p=Path(file); text=p.read_text(); missing=[]
 for link in re.findall(r'\[[^\]]*\]\(([^)]+)\)',text):
  path=link.split('#')[0]
  if path and not re.match(r'[a-z]+:',path) and not (p.parent/path).exists():missing.append(link)
 assert not missing,(file,missing)
 print(f'{file}: {len(text.splitlines())} lines; local Markdown links exist')
assert len(Path('model/DESIGN.md').read_text().splitlines())<=1000
