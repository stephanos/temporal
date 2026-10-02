#!/usr/bin/env bash
# Fetch the Stainless standalone release into UMPIRE_SCALA_TOOLS (default /tmp/umpire-scala-tools).
# It bundles its own Scala front end, Z3 and cvc5, so nothing else is installed.
set -euo pipefail
tools="${UMPIRE_SCALA_TOOLS:-/tmp/umpire-scala-tools}"
version=0.10.2
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) asset=mac-arm64 ;;
  Darwin-x86_64) asset=mac-x64 ;;
  Linux-*) asset=linux ;;
  *) echo "tools.sh: no Stainless build for $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac
if [[ -x "$tools/stainless/stainless" ]]; then
  echo "$tools/stainless/stainless"
  exit 0
fi
mkdir -p "$tools"
zip="stainless-dotty-standalone-$version-$asset.zip"
gh release download "v$version" -R epfl-lara/stainless -p "$zip" -D "$tools" --clobber
unzip -q -o "$tools/$zip" -d "$tools/stainless"
echo "$tools/stainless/stainless"
