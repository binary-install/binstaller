package shell

import _ "embed"

// unifiedScriptTemplate is the unified template for both installer and runner scripts.
// It uses sub-templates to handle differences between installer and runner modes.
//
//go:embed template.tmpl.sh
var unifiedScriptTemplate string

// shlib contains the library of POSIX shell functions.
// Adapted from https://github.com/client9/shlib
//
//go:embed shlib.sh
var shlib string

/*
shlib.sh generation command
cat \
  license.sh \
  is_command.sh \
  echoerr.sh \
  log.sh \
  uname_os.sh \
  uname_arch.sh \
  uname_os_check.sh \
  uname_arch_check.sh \
  license_end.sh | \
  grep -v '^#' | grep -v ' #' | tr -s '\n'
*/

// --- Custom functions ---

//go:embed hash_sha512.sh
var hashSHA512 string

//go:embed hash_sha256.sh
var hashSHA256 string

//go:embed hash_sha1.sh
var hashSHA1 string

//go:embed hash_md5.sh
var hashMD5 string

//go:embed shell_functions.sh
var shellFunctions string
