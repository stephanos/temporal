#!/bin/sh
# fn-108 size comparison for R1. Read-only: reads two listings and writes only to
# stdout/stderr.
#
# usage: size-compare.sh BASELINE_LISTING CURRENT_LISTING
# Both listings are outputs of `SIZE_COUNT_DETAIL=1 sh size-count.sh` under the
# same counting rule. Exit status 0 means the R1 size condition holds, 1 means it
# does not, 2 means the inputs are unusable.
#
# The residual is the production-go change plus the growth of every file in
# overlay-go, generated-go, protocol-input and non-Markdown `other` that grew or
# is new. Growth is summed per file, so a reduction in one such file never
# cancels code moved into another, and code moved out of production into one of
# those classes earns nothing. R1 holds when the residual is negative in both
# code lines and code bytes and no file changed class.
set -eu
LC_ALL=C
export LC_ALL

if [ "$#" -ne 2 ]; then
	echo "usage: size-compare.sh BASELINE_LISTING CURRENT_LISTING" >&2
	exit 2
fi
[ -f "$1" ] || { echo "size-compare: missing baseline listing: $1" >&2; exit 2; }
[ -f "$2" ] || { echo "size-compare: missing current listing: $2" >&2; exit 2; }

awk '
function unusable(msg) {
	print "size-compare: " msg > "/dev/stderr"
	bad = 1
	exit 2
}

function bucket(class, path) {
	if (class == "other") return (path ~ /\.md$/) ? "other-markdown" : "other-non-markdown"
	return class
}

function growth(value) {
	return value > 0 ? value : 0
}

FNR == 1 {
	side++
	in_files = 0
	if ($0 !~ /^# fn-108 size-count, counting rule /) unusable(FILENAME " is not a size-count listing")
	rule[side] = $0
	next
}

/^# files: / { in_files = 1; seen_files[side] = 1; next }

in_files && NF == 5 {
	path = $5
	if (side == 1) {
		base_class[path] = $1
	} else {
		current_class[path] = $1
	}
	b = bucket($1, path)
	path_bucket[side, path] = b
	path_code[side, path] = $3
	path_bytes[side, path] = $4
	files[side, b]++
	physical[side, b] += $2
	code[side, b] += $3
	bytes[side, b] += $4
}

END {
	if (bad) exit 2
	if (side != 2 || !seen_files[1] || !seen_files[2]) {
		print "size-compare: both inputs must be SIZE_COUNT_DETAIL=1 listings" > "/dev/stderr"
		exit 2
	}
	if (rule[1] != rule[2]) {
		print "size-compare: the listings use different counting rules" > "/dev/stderr"
		exit 2
	}
	n = split("production-go overlay-go generated-go protocol-input other-non-markdown other-markdown test-go", order, " ")
	print "# fn-108 size-compare: current minus baseline"
	printf "%-20s %7s %9s %9s %10s\n", "class", "files", "physical", "code", "codebytes"
	for (i = 1; i <= n; i++) {
		c = order[i]
		printf "%-20s %+7d %+9d %+9d %+10d\n", c, files[2, c] - files[1, c], \
			physical[2, c] - physical[1, c], code[2, c] - code[1, c], bytes[2, c] - bytes[1, c]
	}

	changed = 0
	for (path in current_class) {
		if ((path in base_class) && base_class[path] != current_class[path]) {
			changes[++changed] = sprintf("%s: %s -> %s", path, base_class[path], current_class[path])
		}
	}
	print ""
	printf "files that changed class: %d\n", changed
	# Sorted so the report does not depend on awk hash order.
	for (i = 2; i <= changed; i++) {
		line = changes[i]
		for (j = i - 1; j >= 1 && changes[j] > line; j--) changes[j + 1] = changes[j]
		changes[j + 1] = line
	}
	for (i = 1; i <= changed; i++) print "  " changes[i]

	residual_code = code[2, "production-go"] - code[1, "production-go"]
	residual_bytes = bytes[2, "production-go"] - bytes[1, "production-go"]
	m = split("overlay-go generated-go protocol-input other-non-markdown", offsets, " ")
	for (i = 1; i <= m; i++) offsetting[offsets[i]] = 1
	# A path absent from the baseline listing contributes its whole size.
	for (path in current_class) {
		c = path_bucket[2, path]
		if (!(c in offsetting)) continue
		grown_code[c] += growth(path_code[2, path] - path_code[1, path])
		grown_bytes[c] += growth(path_bytes[2, path] - path_bytes[1, path])
	}
	print ""
	print "offsetting growth, summed over files that grew or are new:"
	for (i = 1; i <= m; i++) {
		c = offsets[i]
		printf "  %-20s code %+d codebytes %+d\n", c, grown_code[c], grown_bytes[c]
		residual_code += grown_code[c]
		residual_bytes += grown_bytes[c]
	}
	print ""
	printf "residual (production change plus offsetting growth): code %+d codebytes %+d\n", \
		residual_code, residual_bytes
	if (residual_code < 0 && residual_bytes < 0 && changed == 0) {
		print "R1 size condition: PASS"
		exit 0
	}
	print "R1 size condition: FAIL"
	exit 1
}
' "$1" "$2"
