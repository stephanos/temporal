import json,signal,subprocess,tempfile,unittest
from pathlib import Path
import watchdog as w
class FakeChild:
 pid=987654
 def __init__(self,normal=False): self.code=None;self.signals=[];self.waited=False;self.normal=normal
 def poll(self): return self.code
 def send_signal(self,s): self.signals.append(s)
 def wait(self,timeout=None):
  if timeout is not None and not self.normal and not self.signals:raise subprocess.TimeoutExpired('fake',timeout)
  self.waited=True;self.code=0 if self.normal else -15;return self.code
class Controls(unittest.TestCase):
 def run_case(self,free,precheck=lambda:{},normal=False):
  tmp=tempfile.TemporaryDirectory();self.addCleanup(tmp.cleanup);out=Path(tmp.name)
  child=FakeChild(normal);calls=[];values=iter(free)
  def spaces():return [{'path':'fake','st_dev':123,'free':next(values)*w.GIB}]
  def launch(*a,**k):calls.append(a);return child
  r=w.supervise(['fake-proof'],out,lambda:{'validation_accepted':True},spaces,precheck,interval=.001,popen=launch)
  return r,child,calls
 def test_admission(self):
  r,c,calls=self.run_case([49]);self.assertTrue(r['rejected']);self.assertFalse(calls);self.assertFalse(r['accepted'])
 def test_runtime_waits_terminal(self):
  r,c,calls=self.run_case([50,29]);self.assertEqual(c.signals,[signal.SIGTERM]);self.assertTrue(c.waited);self.assertTrue(r['terminal']);self.assertTrue(r['rejected']);self.assertFalse(r['accepted']);self.assertEqual(r['exit_code'],-15)
 def test_normal(self):
  r,c,calls=self.run_case([50,31],normal=True);self.assertTrue(r['accepted']);self.assertTrue(r['terminal']);self.assertEqual(c.signals,[])
 def test_source_and_hash_rejection(self):
  for reason in ['source changed','preservation hash mismatch','native hash mismatch']:
   with self.subTest(reason=reason):
    def fail():raise AssertionError(reason)
    r,c,calls=self.run_case([],precheck=fail);self.assertTrue(r['rejected']);self.assertIn(reason,r['rejection']);self.assertFalse(calls)
if __name__=='__main__':
 suite=unittest.defaultTestLoader.loadTestsFromTestCase(Controls)
 result=unittest.TextTestRunner(verbosity=2).run(suite)
 w.write(Path(__file__).parent/'controls.json',{'success':result.wasSuccessful(),'tests_run':result.testsRun,'failures':len(result.failures),'errors':len(result.errors)})
 raise SystemExit(not result.wasSuccessful())
