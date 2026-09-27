# Sourced by mise (see mise.toml [env]); it must only export variables.
#
# The lean4 toolchain's bin directory ships a `clang` that cannot find the macOS SDK headers
# (`stddef.h not found`), and mise puts it on PATH ahead of the system one. cgo resolves its
# default `clang` through PATH, so pin the system clang on macOS. An explicit CC still wins, and
# Linux keeps Go's own default. The Makefile does the same for `make` via xcrun.
if [ -z "${CC:-}" ] && [ "$(uname -s)" = Darwin ]; then
  export CC=/usr/bin/clang
fi
