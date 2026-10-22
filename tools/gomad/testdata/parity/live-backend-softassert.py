#!/usr/bin/env python3
"""Execute transferred backend roots 16 and 17 with the original softassert guest."""
from pathlib import Path
import json
import os
import signal
import threading
import runpy
import subprocess

common = runpy.run_path(str(Path(__file__).with_name('live-backend-common.py')))
Lane, equal = common['Lane'], common['equal']
MODULE, REPOSITORY = common['MODULE'], common['REPOSITORY']


def observation(report_only):
    return dict(messages=['invalid fixture state'], report_only=report_only, inconclusive=False)


def check_reports(rows, report_only, seeds):
    reports = [row for row in rows if row.get('soft_assertions') is not None]
    equal([row['seed'] for row in reports], seeds, 'soft_assertions.seed_order')
    equal([row['soft_assertions'] for row in reports], [observation(report_only)] * len(seeds),
          'soft_assertions.reports')
    return reports


def main():
    lane = Lane(__doc__)
    installation = lane.installation('softassert', cooperative=False)
    target = ['go-test', './tests/gomadfunctional/testdata/softassert', '--', '-test.v']
    cells = []
    for mode in ('own', 'late'):
        for report_only in (False, True):
            name = mode + ('-report-only' if report_only else '-failure')
            flags = ['--build-tag=test_dep'] + (['--soft-assertion-report-only'] if report_only else [])
            root, output = lane.explore(name, [*target, '-assertion-mode=' + mode], REPOSITORY,
                flags, installation=installation, seeds='7,19', execution='300s', overall='600s', keep='all',
                success_limit=2, success_bytes=128 << 20)
            checked = lane.authenticate(root)
            report = lane.report(output)
            equal(report['attempted'], 2, name + '.attempted')
            equal(report['failures'], 0 if report_only else 2, name + '.failures')
            artifacts = lane.artifacts(root)
            equal(len(artifacts), 2 if report_only else 1, name + '.static_deduplication')
            rows = [json.loads(line) for line in output.stdout.splitlines()]
            check_reports(rows, report_only, [7, 19])
            for index, artifact in enumerate(artifacts):
                record = common['seed']['authenticate_artifact'](artifact)
                inputs = json.loads((artifact / 'target/provenance.json').read_bytes())
                equal(inputs['inputs']['soft_assertion_report_only'], report_only, name + '.policy_identity')
                equal(json.loads((artifact / 'evidence/soft-assertions.json').read_bytes()),
                      observation(report_only), name + '.observation')
                equal(b'PASS' in (artifact / 'output/stdout').read_bytes(), True, name + '.guest_passed')
                equal((record['outcome']['termination'], record['outcome']['exit_code']),
                      ('exit', 0), name + '.actual_passing_exit')
                equal(record['outcome']['class'], 'success' if report_only else 'target_failure',
                      name + '.classification')
                if not report_only:
                    equal(record['outcome']['reason'], 'soft_assertion', name + '.reason')
                lane.replay(name + '-replay-' + str(index), artifact, exact=False, failure=not report_only)
            if report_only:
                root, output = lane.explore(name + '-unretained', [*target, '-assertion-mode=' + mode],
                    REPOSITORY, flags, installation=installation, seeds='7,19', execution='300s', overall='600s')
                equal(lane.report(output)['succeeded'], 2, name + '.discarded_successes')
                equal(lane.artifacts(root), [], name + '.discarded_artifacts')
                check_reports([json.loads(line) for line in output.stdout.splitlines()], True, [7, 19])
                root, output = lane.explore(name + '-capacity', [*target, '-assertion-mode=' + mode],
                    REPOSITORY, flags, status=(3,), installation=installation, seeds='7,19',
                    execution='300s', overall='600s', output=8)
                error = json.loads(output.stdout.splitlines()[-1])
                equal(error['classification'], 'capacity', name + '.capacity')
                equal(error['soft_assertions']['inconclusive'], True, name + '.inconclusive')
                equal(error['soft_assertions']['report_only'], True, name + '.capacity_policy')
                equal(list(root.glob('evidence/*/manifest.json')), [], name + '.capacity_no_artifacts')
            cells.append(dict(mode=mode, report_only=report_only, seeds=[7, 19]))
    lane.roots.append(dict(root=16, original='TestSoftAssertionActualPassingGuestArtifactReplay', cells=cells))

    # Read each notification while the child is running: inspection must already
    # expose the corresponding committed outcome when the report is delivered.
    installation = lane.installation('softassert-choice')
    destination = lane.args.work / 'choice-committed'
    command = [lane.args.binary, '--installation', installation, 'explore', '--json',
        '--strategy=choice-exploration', '--seeds=7', '--parallel=1', '--choices',
        '--choice-bytes=1048576', '--max-executions=2', '--max-choice-depth=4',
        '--max-exploration-bytes=8388608', '--on-failure=all', '--failure-budget=1',
        '--coverage=choice', '--execution-timeout=300s', '--overall-timeout=600s',
        '--terminate-grace=100ms', '--output-limit=1048576', '--world-transition-limit=1048576',
        '--keep-successes=none', '--soft-assertion-report-only', '--build-tag=test_dep',
        '--artifacts', destination, '--working-dir', REPOSITORY, *target, '-assertion-mode=late']
    reports, lines = [], []
    acknowledgments = lane.args.work / 'soft-report-ack'
    acknowledgments.mkdir(mode=0o700)
    stderr_path = lane.args.work / 'choice-committed.stderr'
    with stderr_path.open('wb') as stderr:
        child = subprocess.Popen(list(map(str, command)), stdout=subprocess.PIPE, stderr=stderr,
                                 env=dict(lane.environment, GOMAD_FIXTURE_SOFT_REPORT_ACK=str(acknowledgments)),
                                 start_new_session=True)
        def terminate():
            if child.poll() is None:
                try:
                    os.killpg(child.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
        deadline = threading.Timer(1200, terminate)
        deadline.daemon = True
        deadline.start()
        try:
            for line in child.stdout:
                lines.append(line)
                row = json.loads(line)
                if row.get('soft_assertions') is not None:
                    root = Path(row['campaign_path'])
                    inspected = json.loads(lane.run('choice-progress-inspect-' + str(len(reports)),
                        ['inspect', '--json', root]).stdout)
                    executions = inspected['campaign']['executions']
                    equal(len(executions) > len(reports), True, 'choice.progress_after_durable_outcome')
                    equal(row['soft_assertions'], observation(True), 'choice.progress_observation')
                    (acknowledgments / str(len(reports))).write_bytes(b'committed outcome inspected\n')
                    reports.append(row)
            status = child.wait(timeout=600)
        finally:
            deadline.cancel()
            if child.poll() is None:
                terminate()
                child.wait()
    (lane.args.work / 'choice-committed.stdout').write_bytes(b''.join(lines))
    lane.commands.append(dict(name='choice-committed', argv=list(map(str, command)), status=status,
                              expected_statuses=[0],
                              environment_overrides=dict(GOMAD_FIXTURE_SOFT_REPORT_ACK=str(acknowledgments)),
                              stdout_sha256=common['digest'](b''.join(lines)),
                              stderr_sha256=common['digest'](stderr_path.read_bytes())))
    (lane.args.work / 'commands.json').write_bytes(common['encode'](lane.commands))
    equal(status, 0, 'choice.status')
    result = next(json.loads(line) for line in reversed(lines) if json.loads(line).get('type') == 'complete')
    equal(result['attempted'] > 0, True, 'choice.attempted')
    equal((len(reports), result['failures']), (result['attempted'], 0), 'choice.report_count')
    root = Path(result['campaign'])
    lane.authenticate(root, choice=True)
    inspected = json.loads(lane.run('choice-final-inspect', ['inspect', '--json', root]).stdout)
    executions = inspected['campaign']['executions']
    equal(len(executions), len(reports), 'choice.inspected_report_count')
    for execution, report in zip(executions, reports):
        equal(execution['soft_assertions'], report['soft_assertions'], 'choice.committed_observation')
        equal(execution['artifact'], None, 'choice.no_success_artifact')
        equal((execution['record']['outcome']['class'], execution['record']['outcome']['exit_code']),
              ('success', 0), 'choice.guest_passed')
    lane.roots.append(dict(root=17, original='TestSoftAssertionChoiceCampaignReportsCommittedRounds',
                           reports=len(reports)))
    lane.finish()


if __name__ == '__main__':
    main()
