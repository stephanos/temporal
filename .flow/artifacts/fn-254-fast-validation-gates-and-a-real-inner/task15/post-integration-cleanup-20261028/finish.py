"""Prepared continuation of exactly four successful tagged cleans; fixed private remainder only."""
import argparse
import hashlib
import importlib.util
import json
import os
import re
import shutil
import stat
import sys
import uuid
from pathlib import Path

HERE = Path(__file__).resolve().parent
PRIOR = HERE/'cleanup-execution.json'
PRIOR_SHA = 'b98ad385f08dcfa8bcf68c2e5184f475ad80e7c118fae4cedaf35eefaa762d3a'
CLEANUP_SHA = '8ccd5dfc80ace0263379ba09da2fa9584d10e61fdf0aedff0c9fd08040a5860f'
PRIMARY_SHA = '68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610'
SUPPLEMENTAL_SHA = 'e22efa29f106d7a2f8bf96374b080ee8d5c67b73b1dc4adc864daf7b9554a06a'
CLASSIFICATION_SHA = '78fa352ec05730c873597c1b928d80734e779884861dbc1a2915181997d6e9a5'
TOOLS_SHA = '8b11813eda5a3a5482ca065f8603e57fdaf43b946b3db40acc67345128550de2'
TARGET = Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target')


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024*1024), b''):
            value.update(chunk)
    return value.hexdigest()


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def check_logs_and_pids(prior, expected_commands, directory=HERE, proc=Path('/proc')):
    assert len(prior['commands']) == len(expected_commands) == 4
    assert [row['command'] for row in prior['commands']] == expected_commands, 'prior commands differ'
    result = []
    for row in prior['commands']:
        assert row['terminal'] is True and row['exit_code'] == 0, 'prior clean did not finish successfully'
        pid = row['pid']
        assert type(pid) is int and pid > 0 and not (proc/str(pid)).exists(), 'prior Cargo PID still exists'
        logs = {}
        for name in ['stdout','stderr']:
            path = Path(row[name])
            assert path.parent == directory and path.name.startswith('cleanup-') and path.name.endswith('.'+name+'.log')
            assert not path.is_symlink() and stat.S_ISREG(path.lstat().st_mode), 'prior log is not regular'
            actual = digest(path)
            assert actual == row[name+'_sha256'], 'prior log hash changed'
            logs[name] = actual
        result.append(dict(pid=pid, absent=True, terminal=True, exit_code=0, logs=logs))
    return result


def execute(options):
    # All mutable operation paths come from the sealed original helper, whose literal target is checked below.
    assert digest(HERE/'cleanup.py') == CLEANUP_SHA
    helper = load('finish_cleanup_helper', HERE/'cleanup.py')
    output = (options.receipt or HERE/('finish-'+uuid.uuid4().hex+'.json')).resolve()
    assert output.parent == HERE and output.name.startswith('finish-') and not output.is_symlink()
    receipt = dict(schema='fn25415-private-cache-finish-v1', prepared_script_sha256=digest(Path(__file__)),
                   prior_receipt=str(PRIOR), prior_receipt_sha256=PRIOR_SHA, original_cleanup_sha256=CLEANUP_SHA,
                   target=str(TARGET), started_utc=helper.timestamp(), mutation_started=False, complete=False,
                   free_before=helper.free_space(), allocated_before=helper.allocated(TARGET))
    helper.write_receipt(output, receipt, initial=True)
    supplement = None
    protected = None
    prior = None
    inspector = None
    failure = None
    try:
        assert options.expected_script_sha256 == receipt['prepared_script_sha256'], 'finish source differs from reviewed SHA'
        assert options.expected_prior_sha256 == PRIOR_SHA and digest(PRIOR) == PRIOR_SHA, 'prior receipt changed'
        assert options.expected_cleanup_sha256 == CLEANUP_SHA
        assert options.expected_primary_sha256 == PRIMARY_SHA
        assert options.expected_supplemental_sha256 == SUPPLEMENTAL_SHA
        assert options.expected_classification_sha256 == CLASSIFICATION_SHA
        assert options.expected_tools_sha256 == TOOLS_SHA
        assert helper.TARGET == TARGET and helper.PRIMARY_SHA == PRIMARY_SHA and helper.SUPPLEMENTAL_SHA == SUPPLEMENTAL_SHA
        assert digest(HERE/'tooling-manifest.json') == TOOLS_SHA
        tooling = json.loads((HERE/'tooling-manifest.json').read_text())
        for name, checksum in tooling['files'].items():
            assert digest(HERE/name) == checksum, 'sealed classification/tooling changed'
        assert digest(HERE/'classification.json') == CLASSIFICATION_SHA
        census = json.loads((HERE/'classification.json').read_text())
        prior = json.loads(PRIOR.read_text())
        assert prior['kind'] == 'root-authorized-exact-private-cache-cleanup'
        assert prior['target'] == str(TARGET) and prior['script_sha256'] == CLEANUP_SHA
        assert prior['mutation_started'] is True and prior['complete'] is False and prior['private_target_absent'] is False
        assert prior['failure']['type'] == 'AssertionError' and prior['failure']['message'] == 'unrecognized unreadable process'
        assert 'remainder_action' not in prior, 'prior run already attempted root remainder removal'
        assert prior['protected_exact_checks_passed'] is True and prior['protected_after'] == prior['protected_before']
        assert prior['custody_after']['verified'] is True and prior['custody_after']['supplemental_unique_members'] == 10708
        assert prior['custody_after']['primary_verification']['archive_members'] == 4801
        assert prior['custody_after']['primary_verification']['manifest_sha256'] == PRIMARY_SHA
        commands = [row['command'] for row in census['proposed_tagged_cargo_clean']]
        receipt['prior_commands_verified'] = check_logs_and_pids(prior, commands)
        target_inode = helper.guard_target()
        supplement = load('finish_supplement', HERE/'supplemental.py')
        inspector = load('finish_inspector', HERE/'classify.py')
        collector, _ = supplement.helpers()
        supplement.validate_map(census, collector)
        receipt['ownership_before'] = helper.ownership(inspector, options.accept_recorded_daemon_visibility_limits)
        receipt['custody_before'] = helper.verify_custody(supplement)
        protected = prior['protected_before']
        receipt['protected_original_snapshot_matches'] = helper.protected_snapshot() == protected
        assert receipt['protected_original_snapshot_matches'], 'original protected snapshot changed'
        receipt['remaining_subset_before'] = helper.verify_remaining(TARGET, census['all_files'])
        helper.guard_target(target_inode)
        # The expensive hashes/custody reads must not leave an unobserved ownership interval.
        receipt['ownership_after_hashes'] = helper.ownership(inspector, options.accept_recorded_daemon_visibility_limits)
        assert helper.protected_snapshot() == protected
        assert digest(PRIOR) == PRIOR_SHA, 'prior receipt changed during admission'
        receipt['prior_commands_verified_again'] = check_logs_and_pids(prior, commands)
        helper.guard_target(target_inode)
        receipt['ownership_immediately_before_removal'] = helper.ownership(inspector, options.accept_recorded_daemon_visibility_limits)
        assert shutil.rmtree.avoids_symlink_attacks, 'fd-based symlink-resistant removal required'
        receipt['mutation_started'] = True
        receipt['remainder_action'] = dict(method='shutil.rmtree', exact_literal=str(TARGET), wildcard=False, force_tag=False,
                                           started_utc=helper.timestamp(), no_cargo_commands_repeated=True)
        helper.write_receipt(output, receipt)
        shutil.rmtree(TARGET)
        receipt['remainder_action']['ended_utc'] = helper.timestamp()
        assert not os.path.lexists(TARGET), 'private remainder still exists'
    except BaseException as error:
        failure = error
        receipt['failure'] = dict(type=type(error).__name__, message=str(error), at=helper.timestamp())
    finally:
        receipt['free_after'] = helper.free_space()
        receipt['allocated_after'] = helper.allocated(TARGET)
        receipt['private_target_absent'] = not os.path.lexists(TARGET)
        if receipt['mutation_started'] and supplement is not None:
            try:
                receipt['custody_after'] = helper.verify_custody(supplement)
                receipt['ownership_after'] = helper.ownership(inspector, options.accept_recorded_daemon_visibility_limits)
            except BaseException as error:
                receipt['post_mutation_verification_failure'] = dict(type=type(error).__name__, message=str(error))
                failure = failure or error
        if protected is not None:
            try:
                receipt['protected_original_snapshot_matches_after'] = helper.protected_snapshot() == protected
                assert receipt['protected_original_snapshot_matches_after'], 'protected source/proof/CAS/parent identity changed'
            except BaseException as error:
                receipt['protected_failure'] = dict(type=type(error).__name__, message=str(error))
                failure = failure or error
        try:
            assert digest(PRIOR) == PRIOR_SHA, 'prior receipt was modified'
            receipt['original_partial_receipt_unchanged'] = True
            if prior is not None:
                receipt['prior_logs_and_pid_absence_after'] = check_logs_and_pids(prior, [row['command'] for row in prior['commands']])
        except BaseException as error:
            receipt['prior_evidence_failure'] = dict(type=type(error).__name__, message=str(error))
            failure = failure or error
        receipt['complete'] = failure is None and receipt['private_target_absent'] and receipt.get('custody_after',{}).get('verified',False) and receipt.get('protected_original_snapshot_matches_after',False)
        receipt['ended_utc'] = helper.timestamp()
        receipt['visibility_limit'] = 'Only the original three acknowledged foreign daemon limitations are permitted. Unknown unreadable processes or readable private handles always refuse.'
        receipt['free_space_limit'] = 'Distinct source and /tmp devices sampled; active task12 parent contents are protected, not asserted idle or byte-stable.'
        helper.write_receipt(output, receipt)
        print(json.dumps(dict(receipt=str(output), complete=receipt['complete'], mutation_started=receipt['mutation_started'], private_target_absent=receipt['private_target_absent']), indent=2), flush=True)
    return 0 if receipt['complete'] else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--execute', action='store_true')
    for name in ['script','prior','cleanup','primary','supplemental','classification','tools']:
        parser.add_argument('--expected-'+name+'-sha256')
    parser.add_argument('--accept-recorded-daemon-visibility-limits', action='store_true')
    parser.add_argument('--receipt', type=Path)
    options = parser.parse_args()
    if not options.execute:
        print(json.dumps(dict(prepared_only=True, mutation_started=False, schema='fn25415-private-cache-finish-v1', target=str(TARGET))))
        return 0
    for name in ['script','prior','cleanup','primary','supplemental','classification','tools']:
        assert re.fullmatch('[0-9a-f]{64}', getattr(options,'expected_'+name+'_sha256') or ''), 'expected '+name+' SHA required'
    return execute(options)


if __name__ == '__main__':
    sys.exit(main())
