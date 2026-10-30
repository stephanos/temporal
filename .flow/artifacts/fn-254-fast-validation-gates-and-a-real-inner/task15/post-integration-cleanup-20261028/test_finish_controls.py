"""Tiny prior-command evidence controls; no real process, target, Cargo or cleanup mutation."""
import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
import finish

HERE = Path(__file__).resolve().parent


class Controls(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(dir=HERE, prefix='safe-finish-evidence-')
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.proc = self.root/'fake-proc'
        self.proc.mkdir()
        self.commands = [['fake-cargo', 'clean', '--target-dir', '/fake/'+str(index)] for index in range(4)]
        self.prior = dict(commands=[])
        for index, command in enumerate(self.commands):
            row = dict(command=command, pid=90000+index, terminal=True, exit_code=0)
            for name in ['stdout','stderr']:
                path = self.root/('cleanup-fake-'+str(index)+'.'+name+'.log')
                path.write_bytes(b'fixture')
                row[name] = str(path)
                row[name+'_sha256'] = hashlib.sha256(b'fixture').hexdigest()
            self.prior['commands'].append(row)

    def check(self, prior=None):
        return finish.check_logs_and_pids(prior or self.prior, self.commands, self.root, self.proc)

    def test_four_exact_terminal_zero_commands_are_admitted(self):
        self.assertEqual(len(self.check()), 4)
        self.assertTrue(all(row['absent'] for row in self.check()))

    def test_failed_or_unfinished_commands_are_refused(self):
        for key, value in [('terminal', False), ('exit_code', 37)]:
            prior = copy.deepcopy(self.prior)
            prior['commands'][1][key] = value
            with self.subTest(key=key), self.assertRaisesRegex(AssertionError, 'did not finish successfully'):
                self.check(prior)
        prior = copy.deepcopy(self.prior)
        prior['commands'][0]['command'] = ['other']
        with self.assertRaisesRegex(AssertionError, 'commands differ'):
            self.check(prior)

    def test_present_prior_pid_is_refused(self):
        (self.proc/str(self.prior['commands'][0]['pid'])).mkdir()
        with self.assertRaisesRegex(AssertionError, 'PID still exists'):
            self.check()

    def test_changed_or_redirected_logs_are_refused(self):
        path = Path(self.prior['commands'][0]['stdout'])
        path.write_bytes(b'changed')
        with self.assertRaisesRegex(AssertionError, 'log hash changed'):
            self.check()
        path.write_bytes(b'fixture')
        path.unlink()
        path.symlink_to(Path(self.prior['commands'][1]['stdout']))
        with self.assertRaisesRegex(AssertionError, 'log is not regular'):
            self.check()


if __name__ == '__main__':
    result = unittest.TextTestRunner(verbosity=2).run(unittest.defaultTestLoader.loadTestsFromTestCase(Controls))
    output = dict(success=result.wasSuccessful(), tests_run=result.testsRun, failures=len(result.failures), errors=len(result.errors),
                  finish_sha256=hashlib.sha256((HERE/'finish.py').read_bytes()).hexdigest(),
                  real_cleanup_executed=False, real_cargo_executed=False,
                  scope='tiny fake logs and fake /proc tree only')
    (HERE/'finish-controls.json').write_text(json.dumps(output,indent=2)+'\n')
    raise SystemExit(not result.wasSuccessful())
