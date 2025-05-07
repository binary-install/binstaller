hash_md5() {
  target=${1:-/dev/stdin}
  if is_command md5sum; then
    sum=$(md5sum "$target" 2>/dev/null) || return 1
    echo "$sum" | cut -d ' ' -f 1
  elif is_command md5; then
    md5 -q "$target" 2>/dev/null
  else
    log_crit "hash_md5 unable to find command to compute md5 hash"
    return 1
  fi
}

hash_verify() {
  hash_verify_internal "$1" "$2" hash_md5
}
