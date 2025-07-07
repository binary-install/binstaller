untar() {
  tarball=$1
  strip_components=${2:-0} # default 0
  case "${tarball}" in
  *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar.xz) tar --no-same-owner -xJf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar.bz2) tar --no-same-owner -xjf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar) tar --no-same-owner -xf "${tarball}" --strip-components "${strip_components}" ;;
  *.gz) gunzip "${tarball}" ;;
  *.zip)
    # unzip doesn't have a standard --strip-components
    # Workaround: extract to a subdir and move contents up if stripping
    if [ "$strip_components" -gt 0 ]; then
      extract_dir=$(basename "${tarball%.zip}")_extracted
      unzip -q "${tarball}" -d "${extract_dir}"
      # Move contents of the *first* directory found inside extract_dir up
      # This assumes wrap_in_directory=true convention
      first_subdir=$(find "${extract_dir}" -mindepth 1 -maxdepth 1 -type d -print -quit)
      if [ -n "$first_subdir" ]; then
        # Move all contents (* includes hidden files)
        mv "${first_subdir}"/* .
        # Optionally remove the now-empty subdir and the extract_dir
        rmdir "${first_subdir}"
        rmdir "${extract_dir}"
      else
        log_warn "Could not find subdirectory in zip to strip components from ${extract_dir}"
        # Files are extracted in current dir anyway, proceed
      fi
    else
      unzip -q "${tarball}"
    fi
    ;;
  *)
    log_err "untar unknown archive format for ${tarball}"
    return 1
    ;;
  esac
}

extract_hash() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "extract_hash checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  grep -E "([[:space:]]|/|\*)${BASENAME}$" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1
}


hash_verify() {
  TARGET_PATH=$1
  SUMFILE=$2
  if [ -z "${SUMFILE}" ]; then
    log_err "hash_verify checksum file not specified in arg2"
    return 1
  fi
  got=$(hash_compute "$TARGET_PATH")
  if [ -z "${got}" ]; then
    log_err "failed to calculate hash: ${TARGET_PATH}"
    return 1
  fi
  # 1) “hash-only” line?
  if grep -i -E "^${got}[[:space:]]*$" "$SUMFILE" >/dev/null 2>&1; then
    return 0
  fi
  # 2) Check hash & file name match
  want=$(extract_hash "${TARGET_PATH}" "${SUMFILE}")
  if [ "$want" != "$got" ]; then
    log_err "hash_verify checksum for '$TARGET_PATH' did not verify ${want} vs ${got}"
    return 1
  fi
}

# GitHub HTTP download functions with GITHUB_TOKEN support
github_http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -n "$GITHUB_TOKEN" ]; then
    log_debug "Using GITHUB_TOKEN for authentication"
    if [ -z "$header" ]; then
      curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" -o "$local_file" "$source_url"
    else
      curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" -H "$header" -o "$local_file" "$source_url"
    fi
  else
    if [ -z "$header" ]; then
      curl -fsSL -o "$local_file" "$source_url"
    else
      curl -fsSL -H "$header" -o "$local_file" "$source_url"
    fi
  fi
}
github_http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -n "$GITHUB_TOKEN" ]; then
    log_debug "Using GITHUB_TOKEN for authentication"
    if [ -z "$header" ]; then
      wget -q --header "Authorization: Bearer $GITHUB_TOKEN" -O "$local_file" "$source_url"
    else
      wget -q --header "Authorization: Bearer $GITHUB_TOKEN" --header "$header" -O "$local_file" "$source_url"
    fi
  else
    if [ -z "$header" ]; then
      wget -q -O "$local_file" "$source_url"
    else
      wget -q --header "$header" -O "$local_file" "$source_url"
    fi
  fi
}
github_http_download() {
  log_debug "github_http_download $2"
  if is_command curl; then
    github_http_download_curl "$@"
    return
  elif is_command wget; then
    github_http_download_wget "$@"
    return
  fi
  log_crit "github_http_download unable to find wget or curl"
  return 1
}
github_http_copy() {
  tmp=$(mktemp)
  github_http_download "${tmp}" "$@" || return 1
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}
github_release() {
  owner_repo=$1
  version=$2
  test -z "$version" && version="latest"
  giturl="https://github.com/${owner_repo}/releases/${version}"
  json=$(github_http_copy "$giturl" "Accept:application/json")
  test -z "$json" && return 1
  version=$(echo "$json" | tr -s '\n' ' ' | sed 's/.*"tag_name":"//' | sed 's/".*//')
  test -z "$version" && return 1
  echo "$version"
}
