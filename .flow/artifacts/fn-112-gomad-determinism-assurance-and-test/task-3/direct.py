"""Launch a prebuilt target directly with a choice trace, with and without the
diagnostic trace, and compare what the target produced."""
import hashlib, json, os, struct, subprocess, sys

CHOICE = 8 << 20
DIAGNOSTIC = 8 << 20

def launch(binary, arguments, seed, diagnostics):
    for name in ("choices", "diagnostics"):
        if os.path.exists(name):
            os.remove(name)
    choices = os.open("choices", os.O_RDWR | os.O_CREAT, 0o600)
    os.write(choices, b"GOMADCH\x02" + struct.pack(">I", 2) + bytes(4) + struct.pack(">QQ", CHOICE, 64) + bytes(32))
    os.ftruncate(choices, CHOICE)
    reader, writer = os.pipe()
    environment = {"GOMADSEED": str(seed), "GOMAD3_CHOICE_TRACE_FD": str(choices), "GOMAD3_CHOICE_TERMINAL_FD": str(writer), "GOMAD3_CHOICE_TRACE_BYTES": str(CHOICE), "GOMAD3_CHOICE_MODE": "1"}
    descriptors = [choices, writer]
    if diagnostics:
        trace = os.open("diagnostics", os.O_RDWR | os.O_CREAT, 0o600)
        os.write(trace, b"GOMADDG\x01" + struct.pack(">I", 1) + bytes(4) + struct.pack(">QQ", DIAGNOSTIC, 64) + bytes(32))
        os.ftruncate(trace, DIAGNOSTIC)
        environment.update({"GOMAD3_DIAGNOSTIC_TRACE_FD": str(trace), "GOMAD3_DIAGNOSTIC_TRACE_BYTES": str(DIAGNOSTIC)})
        descriptors.append(trace)
    process = subprocess.run([binary] + arguments, env=environment, pass_fds=descriptors, capture_output=True)
    os.close(writer)
    terminal = os.read(reader, 4096)
    os.close(reader)
    for descriptor in descriptors:
        if descriptor != writer:
            os.close(descriptor)
    payload = open("choices", "rb").read()
    next_offset, records = struct.unpack(">QQ", payload[24:40])
    result = {
        "exit": process.returncode,
        "stdout_sha256": hashlib.sha256(process.stdout).hexdigest(),
        "stderr_sha256": hashlib.sha256(process.stderr).hexdigest(),
        "choice_records": records,
        "choice_sha256": hashlib.sha256(payload[64:next_offset]).hexdigest(),
        "choice_terminal_sha256": hashlib.sha256(terminal).hexdigest(),
        "choice_terminal_state": terminal[12] if len(terminal) > 12 else None,
    }
    if diagnostics:
        payload = open("diagnostics", "rb").read()
        next_offset, records = struct.unpack(">QQ", payload[24:40])
        result.update({"diagnostic_records": records, "diagnostic_state": payload[12], "diagnostic_sha256": hashlib.sha256(payload[64:next_offset]).hexdigest()})
    return result

def main():
    binary, seed, repetitions, arguments = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4:]
    off = [launch(binary, arguments, seed, False) for _ in range(repetitions)]
    on = [launch(binary, arguments, seed, True) for _ in range(repetitions)]
    shared = ("exit", "stdout_sha256", "stderr_sha256", "choice_records", "choice_sha256", "choice_terminal_sha256", "choice_terminal_state")
    report = {
        "binary": os.path.basename(binary), "arguments": arguments, "seed": seed, "repetitions": repetitions,
        "off": off[0], "on": on[0],
        "off_repeatable": all(run == off[0] for run in off),
        "on_repeatable": all(run == on[0] for run in on),
        "on_matches_off": all(on[0][key] == off[0][key] for key in shared),
        "one_digest_per_choice_record": on[0]["diagnostic_records"] == on[0]["choice_records"],
    }
    report["pass"] = off[0]["exit"] == 0 and report["off_repeatable"] and report["on_repeatable"] and report["on_matches_off"] and report["one_digest_per_choice_record"] and on[0]["diagnostic_state"] == 1
    json.dump(report, sys.stdout, indent=2, sort_keys=True)
    print()
    sys.exit(0 if report["pass"] else 1)

main()
