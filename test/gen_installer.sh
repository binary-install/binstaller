#!/bin/bash
gen_and_run() {
  BINSTALLER_CONFIG=$1
  TOOL=$(basename "${BINSTALLER_CONFIG%.binstaller.yml}")
  ./binst gen --quiet --config "testdata/${TOOL}.binstaller.yml" -o "testdata/${TOOL}.install.sh"
}

for f in testdata/*.binstaller.yml; do
  gen_and_run "$f"
done
