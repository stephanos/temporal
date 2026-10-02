#!/bin/sh
# fn-108 size accounting (counting rule v2). Read-only: reads the working tree
# and writes only to stdout/stderr. Takes no arguments; run from anywhere inside
# the repository. SIZE_COUNT_DETAIL=1 appends one row per file.
#
# Inventory: every path under the three Gomad directories that git tracks or
# would track (untracked and not ignored), read from the working tree. A new
# file counts before it is committed. The only ignored paths allowed are the two
# build-output roots tools/gomad3/.toolchain/ and tools/gomad3/.bin/; any other
# ignored path could hide source from the count and makes the script fail. The
# rule is stated in baseline.md next to this script.
set -eu
LC_ALL=C
export LC_ALL

if [ "$#" -ne 0 ]; then
	echo "size-count.sh takes no arguments" >&2
	exit 2
fi

top=$(git rev-parse --show-toplevel)
cd "$top"

list=$(git ls-files --cached --others --exclude-standard -- \
	tools/gomad3 tools/gomad3sim tools/gomad3integration | sort -u)

ignored=$(git ls-files --others --ignored --exclude-standard --directory -- \
	tools/gomad3 tools/gomad3sim tools/gomad3integration)

# Refuses an ignored path outside the two build-output roots.
check_ignored() {
	while IFS= read -r path; do
		case "$path" in
		'' | tools/gomad3/.toolchain/ | tools/gomad3/.bin/) ;;
		*)
			echo "size-count: ignored path hides files from the inventory: $path" >&2
			return 1
			;;
		esac
	done
}

printf '%s\n' "$ignored" | check_ignored

# Prints the inventory paths that exist as regular files; refuses anything else.
present_files() {
	while IFS= read -r path; do
		[ -n "$path" ] || continue
		case "$path" in
		*[!A-Za-z0-9._/+@-]*)
			echo "size-count: unsupported character in path: $path" >&2
			return 1
			;;
		esac
		if [ -L "$path" ]; then
			echo "size-count: symbolic link in inventory: $path" >&2
			return 1
		fi
		# A path deleted from the working tree but still in the index is absent.
		[ -e "$path" ] || continue
		if [ ! -f "$path" ]; then
			echo "size-count: not a regular file: $path" >&2
			return 1
		fi
		printf '%s\n' "$path"
	done
}

present=$(printf '%s\n' "$list" | present_files)

printf '%s\n' "$present" | awk -v detail="${SIZE_COUNT_DETAIL:-0}" '
function nonws(c) {
	return c != "" && c != " " && c != "\t" && c != "\r" && c != "\f" && c != "\v"
}

function fail(msg) {
	print "size-count: " msg > "/dev/stderr"
	failed = 1
	exit 1
}

# Go: a code byte is a non-whitespace byte outside // and /* */ comments, except
# a comma or semicolon outside a literal (a trailing comma or an explicit
# semicolon exists or not depending on line layout alone). String, rune and
# raw-string contents are code. A code line carries at least one code byte.
function count_go(path, n,    i, j, len, line, c, c2, q, state, nb, tmp) {
	state = 0 # 0 code, 1 block comment, 2 raw string
	code_lines = 0
	code_bytes = 0
	for (i = 1; i <= n; i++) {
		line = L[i]
		nb = 0
		if (state == 0 && line !~ /[\/"\047`]/) {
			tmp = line
			nb = gsub(/[^ \t\r\f\v,;]/, "", tmp)
		} else {
			len = length(line)
			j = 1
			while (j <= len) {
				c = substr(line, j, 1)
				if (state == 1) {
					if (c == "*" && substr(line, j + 1, 1) == "/") {
						state = 0
						j += 2
					} else {
						j++
					}
					continue
				}
				if (state == 2) {
					if (c == "`") state = 0
					if (nonws(c)) nb++
					j++
					continue
				}
				if (c == "/") {
					c2 = substr(line, j + 1, 1)
					if (c2 == "/") break
					if (c2 == "*") {
						state = 1
						j += 2
						continue
					}
				}
				if (c == "`") {
					state = 2
					nb++
					j++
					continue
				}
				if (c == "\"" || c == "\047") {
					q = c
					nb++
					j++
					while (j <= len) {
						c = substr(line, j, 1)
						if (c == "\\") {
							nb++
							if (nonws(substr(line, j + 1, 1))) nb++
							j += 2
							continue
						}
						if (nonws(c)) nb++
						j++
						if (c == q) break
					}
					continue
				}
				if (nonws(c) && c != "," && c != ";") nb++
				j++
			}
		}
		if (nb > 0) {
			code_lines++
			code_bytes += nb
		}
	}
	if (state != 0) fail("unterminated comment or raw string in " path)
}

# Other text: a line is comment-only when its first non-whitespace bytes are the
# line-comment prefix of its kind; every other non-blank line is a code line.
function count_text(n, prefix,    i, line, tmp) {
	code_lines = 0
	code_bytes = 0
	for (i = 1; i <= n; i++) {
		line = L[i]
		sub(/^[ \t\r\f\v]+/, "", line)
		if (line == "") continue
		if (prefix != "" && index(line, prefix) == 1) continue
		tmp = line
		code_lines++
		code_bytes += gsub(/[^ \t\r\f\v]/, "", tmp)
	}
}

function generated(n,    i) {
	for (i = 1; i <= n; i++) {
		if (L[i] ~ /^\/\/ Code generated .* DO NOT EDIT/) return 1
		if (L[i] ~ /^package[ \t]/) return 0
	}
	return 0
}

function add(scope, class, phys) {
	files[scope, class]++
	physical[scope, class] += phys
	lines[scope, class] += code_lines
	bytes[scope, class] += code_bytes
}

function row(scope, class) {
	printf "%-24s %-22s %6d %9d %9d %10d\n", scope, class, files[scope, class], \
		physical[scope, class], lines[scope, class], bytes[scope, class]
}

BEGIN {
	nscopes = split("tools/gomad3 tools/gomad3sim tools/gomad3integration", scopes, " ")
	nclasses = split("production-go overlay-go test-go generated-go protocol-input other", classes, " ")
	nkinds = split("tmpl patch json s sh make", kinds, " ")
}

{
	path = $0
	if (path == "") next
	scope = ""
	for (s = 1; s <= nscopes; s++) {
		if (index(path, scopes[s] "/") == 1) scope = scopes[s]
	}
	if (scope == "") fail("path outside the inventory: " path)

	n = 0
	while ((status = (getline text < path)) > 0) L[++n] = text
	if (status < 0) fail("cannot read " path)
	close(path)

	kind = ""
	if (path ~ /\.go$/) {
		if (generated(n)) class = "generated-go"
		else if (path ~ /_test\.go$/ || path ~ /\/testdata\//) class = "test-go"
		else if (index(path, "tools/gomad3/toolchain/runtime/overlay/") == 1) class = "overlay-go"
		else class = "production-go"
		count_go(path, n)
	} else if (path ~ /\.tmpl$/) {
		class = "protocol-input"; kind = "tmpl"; count_text(n, "//")
	} else if (path ~ /\.s$/) {
		class = "protocol-input"; kind = "s"; count_text(n, "//")
	} else if (path ~ /\.patch$/) {
		class = "protocol-input"; kind = "patch"; count_text(n, "")
	} else if (path ~ /\.json$/) {
		class = "protocol-input"; kind = "json"; count_text(n, "")
	} else if (path ~ /\.sh$/) {
		class = "protocol-input"; kind = "sh"; count_text(n, "#")
	} else if (path ~ /\.mk$/ || path ~ /\/Makefile$/) {
		class = "protocol-input"; kind = "make"; count_text(n, "#")
	} else {
		class = "other"
		count_text(n, "")
	}

	add(scope, class, n)
	add(scope, "all", n)
	add("total", class, n)
	add("total", "all", n)
	if (kind != "") add("total", "protocol-input:" kind, n)
	total_files++
	if (detail == "1") {
		detail_rows[total_files] = sprintf("%-15s %7d %7d %8d %s", class, n, code_lines, code_bytes, path)
	}
}

END {
	if (failed) exit 1
	if (total_files == 0) {
		print "size-count: empty inventory" > "/dev/stderr"
		exit 1
	}
	print "# fn-108 size-count, counting rule v2 (see baseline.md)"
	printf "%-24s %-22s %6s %9s %9s %10s\n", "scope", "class", "files", "physical", "code", "codebytes"
	for (s = 1; s <= nscopes; s++) {
		for (c = 1; c <= nclasses; c++) row(scopes[s], classes[c])
		row(scopes[s], "all")
	}
	for (c = 1; c <= nclasses; c++) row("total", classes[c])
	row("total", "all")
	for (k = 1; k <= nkinds; k++) row("total", "protocol-input:" kinds[k])
	if (detail == "1") {
		print ""
		print "# files: class physical code codebytes path"
		for (i = 1; i <= total_files; i++) print detail_rows[i]
	}
}
'
