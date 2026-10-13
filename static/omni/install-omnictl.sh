#!/usr/bin/env sh

set -eu

INSTALLPATH=${INSTALLPATH:-/usr/local/bin}
BINARY_NAME=${BINARY_NAME:-omnictl}
REPO="siderolabs/omni"

happyexit() {
  echo "Installed version:"
  echo "$($BINARY_NAME version 2>/dev/null || true)"
  echo ""
  echo "Next steps:"
  echo ""
  echo "  $BINARY_NAME get clusters          # trigger login and list clusters"
  echo ""
  echo "Need more details? See:"
  echo "  https://docs.siderolabs.com/omni/getting-started/install-and-configure-omnictl/"
  echo ""
  exit 0
}

validate_checksum() {
  filename=$1
  url="https://github.com/${REPO}/releases/latest/download/sha256sum.txt"
  SHA=$(curl --proto '=https' --tlsv1.2 -sSfL "${url}" | grep " ${filename}\$" | awk '{print $1 }')
  echo ""
  echo "Validating checksum..."

  case $checksumbin in
    *openssl)
      checksum=$($checksumbin dgst -sha256 "${filename}" | sed -e 's/^.* //')
      ;;
    *shasum)
      checksum=$($checksumbin -a256 "${filename}" | sed -e 's/ .*$//')
      ;;
  esac

  if [ "$checksum" != "$SHA" ]; then
    echo "Checksum validation failed." >&2
    return 1
  fi
  echo "Checksum valid."
  return 0
}

OS=$(uname -s)
arch=$(uname -m)
cli_arch=""

case $OS in
  CYGWIN* | MINGW64*)
    OS=windows-amd64.exe
    ;;
  Darwin | Linux | FreeBSD)
    case $arch in
      x86_64)
        cli_arch=amd64
        ;;
      armv8*)
        cli_arch=arm64
        ;;
      aarch64*)
        cli_arch=arm64
        ;;
      amd64|arm64)
        cli_arch=$arch
        ;;
      *)
        echo "There is no ${BINARY_NAME} $OS support for $arch. Please open an issue with your platform details."
        exit 1
        ;;
    esac
    ;;
  *)
    echo "There is no ${BINARY_NAME} support for $OS/$arch. Please open an issue with your platform details."
    exit 1
    ;;
esac

OS=$(echo "$OS" | tr '[:upper:]' '[:lower:]')

dstfile="${INSTALLPATH}/${BINARY_NAME}"

checksumbin=$(command -v openssl) || checksumbin=$(command -v shasum) || {
  echo "Failed to find checksum binary. Please install openssl or shasum."
  echo ""
  echo "You can also elect to just download/install:"
  echo "  https://github.com/${REPO}/releases/latest/download/${srcfile}"
  echo "into ${dstfile}"
  exit 1
}

if [ -e "${dstfile}" ]; then
  echo ""
  echo "${BINARY_NAME} is already installed 🎉"
  echo ""
  echo "To force re-downloading, delete '${dstfile}' then run me again."
  happyexit
fi

tmpdir=$(mktemp -d /tmp/${BINARY_NAME}.XXXXXX)
srcfile="${BINARY_NAME}-${OS}"
if [ -n "${cli_arch}" ]; then
  srcfile="${srcfile}-${cli_arch}"
fi

url="https://github.com/${REPO}/releases/latest/download/${srcfile}"

(
  cd "$tmpdir"

  echo "Downloading ${srcfile}..."
  curl --proto '=https' --tlsv1.2 -fLO "${url}"
  echo "Download complete!"

  if ! validate_checksum "${srcfile}"; then
    exit 1
  fi
  echo ""
  exit
)

(
  super=''
  if [ ! -w "${INSTALLPATH}" ]; then
    if command -v sudo > /dev/null 2>&1; then
      super='sudo --preserve-env=HOME sh -c'
    elif command -v su > /dev/null 2>&1; then
      super='su -c'
    fi
  fi

  if [ -z "${super}" ]; then
    mv "${tmpdir}/${srcfile}" "${dstfile}"
    chmod +x "${dstfile}"
  else
    ${super} 'mv "'${tmpdir}/${srcfile}'" "'${dstfile}'"'
    ${super} 'chmod +x "'${dstfile}'"'
  fi
)

rm -r "$tmpdir"

echo "${BINARY_NAME} was successfully installed 🎉"
echo ""
happyexit
