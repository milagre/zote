#!/bin/sh
# Container entrypoint. Before launching redis-server, re-resolves every peer's hostname in
# /data/nodes.conf so the node boots with current Pod IPs instead of addresses from prior
# lifecycles. Then starts redis-server with cluster announce options.
set -eu

REDIS_NODES="/data/nodes.conf"

# Rewrite every peer's IP in nodes.conf from its stable headless DNS hostname so we boot with the
# current cluster topology, not IPs from a prior lifecycle. Lines look like:
#   <id> <ip>:<port>@<cport>,<hostname> <flags> ...
# When a hostname is absent or unresolvable we leave the line as-is and let gossip reconcile it.
rewrite_nodes_conf() {
  [ -f "$REDIS_NODES" ] || return 0

  _tmp="${REDIS_NODES}.new"
  : > "$_tmp"
  while IFS= read -r _line || [ -n "$_line" ]; do
    _f2=$(printf '%s' "$_line" | awk '{print $2}')
    _host=$(printf '%s' "$_f2" | awk -F, '{print $2}')
    _old=$(printf '%s' "$_f2" | awk -F'[:,]' '{print $1}')
    if [ -z "$_host" ] || [ -z "$_old" ]; then
      printf '%s\n' "$_line" >> "$_tmp"
      continue
    fi

    _new=$(getent hosts "$_host" 2>/dev/null | awk '{print $1; exit}')
    if [ -z "$_new" ] || [ "$_new" = "$_old" ]; then
      printf '%s\n' "$_line" >> "$_tmp"
      continue
    fi

    _new_f2=$(printf '%s' "$_f2" | awk -F: -v ip="$_new" 'BEGIN{OFS=":"} {$1=ip; print}')
    printf '%s\n' "$_line" | awk -v f2="$_new_f2" '{$2=f2; print}' >> "$_tmp"
  done < "$REDIS_NODES"
  mv "$_tmp" "$REDIS_NODES"
}

rewrite_nodes_conf

set -- redis-server /etc/redis/redis.conf --cluster-announce-ip "$POD_IP"

if [ -n "${POD_NAME:-}" ] &&
  [ -n "${POD_NAMESPACE:-}" ] &&
  [ -n "${REDIS_HEADLESS_SERVICE:-}" ]; then
  ANNOUNCE_HOST="${POD_NAME}.${REDIS_HEADLESS_SERVICE}.${POD_NAMESPACE}.svc.cluster.local"
  set -- "$@" --cluster-announce-hostname "$ANNOUNCE_HOST"
fi

exec "$@"
