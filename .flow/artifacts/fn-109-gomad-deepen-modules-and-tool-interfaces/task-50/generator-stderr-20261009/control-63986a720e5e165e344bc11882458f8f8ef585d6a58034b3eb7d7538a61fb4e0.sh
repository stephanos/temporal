#!/usr/bin/bash
task9_format_output=$("$@")
task9_format_rc=$?
if test "$task9_format_rc" -ne 0; then
    exit "$task9_format_rc"
fi
if test -n "$task9_format_output"; then
    printf '%s\n' "$task9_format_output"
    exit 1
fi
