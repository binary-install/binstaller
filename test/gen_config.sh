#!/bin/bash
set -e
# Test goreleaser source
./binst init --source goreleaser --repo reviewdog/reviewdog -o=testdata/reviewdog.binstaller.yml --sha='7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783'
./binst init --source goreleaser --repo actionutils/sigspy -o=testdata/sigspy.binstaller.yml --sha='3e1c6f32072cd4b8309d00bd31f498903f71c422'
# Test aqua source
./binst init --source aqua --repo zyedidia/micro --output=testdata/micro.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
./binst init --source aqua --repo houseabsolute/ubi --output=testdata/ubi.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test rosetta2
./binst init --source aqua --repo ducaale/xh --output=testdata/xh.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test rosetta2 in version overrides
./binst init --source aqua --repo babarot/git-bump --output=testdata/git-bump.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test empty extension (extension hard coded in template)
./binst init --source aqua --repo Lallassu/gorss --output=testdata/gorss.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Checksum file only contains hash (it does not file name).
./binst init --source aqua --repo EmbarkStudios/cargo-deny --output=testdata/cargo-deny.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Checksum file contains `*<file name>` (binary mode. e.g. sha256sum -b)
./binst init --source aqua --repo int128/kauthproxy --output=testdata/kauthproxy.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test .tar.bz2
./binst init --source aqua --repo xo/xo --output=testdata/xo.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test .gz
./binst init --source aqua --repo tree-sitter/tree-sitter --output=testdata/treesitter.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test AssetWithoutExt
./binst init --source aqua --repo Byron/dua-cli --output=testdata/dua-cli.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test replacement in override (should not merge rule)
./binst init --source aqua --repo SuperCuber/dotter --output=testdata/dotter.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
# Test github source
./binst init --source github --repo haya14busa/bump --output=testdata/bump.binstaller.yml
# Test default bin dir with yq modification
./binst init --source github --repo charmbracelet/gum --output=testdata/gum.binstaller.yml
yq -i '.unpack.strip_components = 1' testdata/gum.binstaller.yml
yq -i '.default_bindir = "./bin"' testdata/gum.binstaller.yml
