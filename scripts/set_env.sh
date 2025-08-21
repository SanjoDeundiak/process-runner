NAME="${1}"

if [ -z "${NAME}" ]; then
  echo "ERROR: NAME is empty" >&2
  exit 1
fi

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo "use source to run the script" >&2
  exit 1
fi

export PRN_ADDRESS="localhost:4321"
export PRN_CA_TLS_CERT=$(cat ca.pem)
export PRN_TLS_KEY=$(cat "${NAME}_key.pem")
export PRN_TLS_CERT=$(cat "${NAME}.pem")
