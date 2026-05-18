#!/bin/sh
# Idempotent cluster creation for the bootstrap Job: exits 0 if the cluster is
# already formed or if create reports an already-initialized topology.
set -eu

HOST_BASE="${REDIS_HEADLESS_NAME:?}"
PORT="${REDIS_CLUSTER_PORT:-6379}"
REP="${REDIS_CLUSTER_REPLICAS:?}"
COUNT="${REDIS_NODE_COUNT:?}"
LAST=$((COUNT - 1))
FIRST="${HOST_BASE}-0.${HOST_BASE}"

wait_ping() {
  _host=$1
  _i=0
  while [ "$_i" -lt 300 ]; do
    if redis-cli -h "$_host" -p "$PORT" PING 2>/dev/null | grep -q PONG; then
      return 0
    fi
    _i=$((_i + 1))
    sleep 1
  done
  echo "timeout waiting for ${_host}" >&2
  exit 1
}

_n=0
while [ "$_n" -le "$LAST" ]; do
  wait_ping "${HOST_BASE}-${_n}.${HOST_BASE}"
  _n=$((_n + 1))
done

info=$(redis-cli -h "$FIRST" -p "$PORT" CLUSTER INFO 2>/dev/null || true)

if echo "$info" | grep -q '^cluster_state:ok'; then
  echo "cluster_state:ok — nothing to do"
  exit 0
fi

assigned=$(echo "$info" | awk -F: '/^cluster_slots_assigned:/{gsub(/\r/,""); gsub(/^ /,""); print $2; exit}')
known=$(echo "$info" | awk -F: '/^cluster_known_nodes:/{gsub(/\r/,""); gsub(/^ /,""); print $2; exit}')

if [ "${assigned:-0}" = "16384" ] && [ "${known:-0}" -eq "$COUNT" ]; then
  echo "cluster already has full slot coverage and node count — nothing to do"
  exit 0
fi

addrs=""
_n=0
while [ "$_n" -le "$LAST" ]; do
  _a="${HOST_BASE}-${_n}.${HOST_BASE}:${PORT}"
  if [ -z "$addrs" ]; then
    addrs="$_a"
  else
    addrs="$addrs $_a"
  fi
  _n=$((_n + 1))
done

set +e
# shellcheck disable=SC2086
out=$(redis-cli --cluster create $addrs --cluster-replicas "$REP" --cluster-yes 2>&1)
rc=$?
set -e

if [ "$rc" -eq 0 ]; then
  printf '%s\n' "$out"
  exit 0
fi

printf '%s\n' "$out" >&2

if echo "$out" | grep -qiE 'already part of|not empty\. Either the node already knows|contains some key in database|Busy Group|\[ERR\] .* not empty'; then
  echo "cluster create skipped: topology already initialized" >&2
  exit 0
fi

exit "$rc"
