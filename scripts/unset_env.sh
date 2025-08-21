if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo "use source to run the script" >&2
  exit 1
fi

unset PRN_ADDRESS
unset PRN_CA_TLS_CERT
unset PRN_TLS_KEY
unset PRN_TLS_CERT
