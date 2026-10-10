import sys
sys.dont_write_bytecode = True
import run as r

inputs = sorted(r.OUTPUT.glob('*.log'))+sorted(r.OUTPUT.glob('*.json'))
assert r.run('final-terminal-inspection',['/usr/bin/python3',str(r.PACKET/'inspect-terminal.py')],r.ROOT,inputs=inputs) == 0
