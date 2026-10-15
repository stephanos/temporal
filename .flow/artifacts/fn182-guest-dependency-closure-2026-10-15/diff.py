import sys
sys.path.insert(0, sys.argv[0].rsplit('/',1)[0])
from closure import *
a = load(sys.argv[1]); b = load(sys.argv[2])
ga, gb = graph(a), graph(b)
ra, rb = reach(ga, roots(a)), reach(gb, roots(b))
gone = sorted(set(map(base, ra)) - set(map(base, rb))); new = sorted(set(map(base, rb)) - set(map(base, ra)))
print("baseline pkgs", len(set(map(base,ra))), "slim pkgs", len(set(map(base,rb))), "removed", len(gone), "added", new)
print("removed src bytes", sum(srcbytes(a[n]) for n in a if base(n) in gone and n==base(n)))
mods = sorted({module(a[n]) for n in gone}); modsleft={module(b[n]) for n in rb}
print("modules fully removed:", [m for m in mods if m not in modsleft])
if "-v" in sys.argv: print("\n".join(gone))
