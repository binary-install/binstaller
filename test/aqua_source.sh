#!/bin/bash
set -e
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT HUP INT TERM

AQUA_REGISTRY_SHA='1436b9b02096f39ace945d9c56adb7a5b11df186'

ignore_tools='
cnappgoat # Not found in standard aqua registry
sigspy # Not found in standard aqua registry
bump # Not found in standard aqua registry
reviewdog-nightly # Not found in standard aqua registry
'

gen_config_and_installer() {
  BINSTALLER_CONFIG=$1
  TOOL="$(basename "${BINSTALLER_CONFIG%.binstaller.yml}")"
  REPO="$(yq .repo < "${BINSTALLER_CONFIG}")"
  printf '%s\n' "${ignore_tools}" | grep -Fq "${TOOL}" && echo "Ignore ${TOOL}" && return 0
  echo "Generating ${TOOL}.binstaller.yml for ${REPO}"
  ./binst init --source aqua --repo "${REPO}" --sha "${AQUA_REGISTRY_SHA}" -o "${TMPDIR}/${TOOL}.binstaller.yml"
  ./binst gen --config="${TMPDIR}/${TOOL}.binstaller.yml" -o "${TMPDIR}/${TOOL}.install.sh"
}

for f in testdata/*.binstaller.yml; do
  gen_config_and_installer "$f"
done

ls "${TMPDIR}"/*.install.sh | rush -j5 -k "{} -b ${TMPDIR}"
