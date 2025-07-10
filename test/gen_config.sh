#!/bin/bash
set -e
# Test goreleaser source
./binst init --source goreleaser --repo reviewdog/reviewdog -o=testdata/reviewdog.binstaller.yml --sha='7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783'
./binst init --source goreleaser --repo actionutils/sigspy -o=testdata/sigspy.binstaller.yml --sha='3e1c6f32072cd4b8309d00bd31f498903f71c422'
./binst init --source goreleaser --repo junegunn/fzf -o=testdata/fzf.binstaller.yml --sha='ce95adc66c27d97b8f1bb56f139b7efd3f53e5c4'
./binst init --source goreleaser --repo k1LoW/gh-setup -o=testdata/gh-setup.binstaller.yml --sha='f59adf10c4c7ed2b673a9f0a96fc1f8a37a735bd'
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
echo '# --- manually added ---' >> testdata/gum.binstaller.yml
yq -i '.unpack.strip_components = 1' testdata/gum.binstaller.yml
yq -i '.default_bindir = "./bin"' testdata/gum.binstaller.yml
yq -i '.default_version = "v0.16.0"' testdata/gum.binstaller.yml
# Add rule for 386 -> i386 mapping
yq -i '.asset.rules += [{"when": {"arch": "386"}, "arch": "i386"}]' testdata/gum.binstaller.yml
# Add rules for BSD OS mappings
yq -i '.asset.rules += [{"when": {"os": "freebsd"}, "os": "Freebsd"}]' testdata/gum.binstaller.yml
yq -i '.asset.rules += [{"when": {"os": "netbsd"}, "os": "Netbsd"}]' testdata/gum.binstaller.yml
yq -i '.asset.rules += [{"when": {"os": "openbsd"}, "os": "Openbsd"}]' testdata/gum.binstaller.yml
# Checksums for gum v0.15.0 and v0.16.0 are already embedded in the config file
# Test GitHub release digest
./binst init --source aqua --repo Songmu/tagpr --output=testdata/tagpr.binstaller.yml --sha='1436b9b02096f39ace945d9c56adb7a5b11df186'
./binst embed-checksums -c ./testdata/tagpr.binstaller.yml  --mode calculate --version v1.7.0
