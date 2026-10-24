"""Filter grep diagnostics only for the author facade's valid tests directory."""

import re
import subprocess
import sys


FACADE_TESTS = re.compile(r"(^|[/\"'`\s=(\[{])author/umpire/tests(?=/|[\"'`\s)\]}]|$)")


def main():
    pattern = sys.argv[1]
    pending = sys.stdin.buffer.read()
    found = False
    while pending:
        path, separator, pending = pending.partition(b"\0")
        if not separator:
            raise ValueError("missing grep path separator")
        number, separator, pending = pending.partition(b"\0")
        if not separator or not number.isdigit():
            raise ValueError("invalid grep line number")
        line, _, pending = pending.partition(b"\n")
        text = line.decode("utf-8", "surrogateescape")
        probe = FACADE_TESTS.sub(lambda match: match[1] + "author/umpire/facade_checks", text)
        bad = True
        if probe != text:
            result = subprocess.run(["grep", "-E", "-q", "--", pattern],
                                    input=probe.encode("utf-8", "surrogateescape") + b"\n")
            if result.returncode not in (0, 1):
                return result.returncode
            bad = result.returncode == 0
        if bad:
            sys.stdout.buffer.write(path + b":" + number + b":" + line + b"\n")
            found = True
    return 1 if found else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f"retired-name filtering failed: {error}", file=sys.stderr)
        sys.exit(2)
