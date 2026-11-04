#!/usr/bin/env bash
# 清理本地环境。
#   scripts/down.sh          删除整个 kind 集群（中间件数据一并删除）
#   scripts/down.sh apps     只卸载业务服务（保留中间件、可观测与数据）
source "$(dirname "$0")/lib.sh"

case "${1:-cluster}" in
  apps)
    for r in $(helm --kube-context "$KUBE_CONTEXT" -n "$NS_APP" list -q); do
      log "helm uninstall $r"
      helm --kube-context "$KUBE_CONTEXT" -n "$NS_APP" uninstall "$r" --wait >/dev/null
    done
    k -n "$NS_APP" delete configmap acme-stack --ignore-not-found >/dev/null
    ok "业务服务已卸载"
    ;;
  cluster)
    if [[ "$CLUSTER_PROVIDER" != "kind" ]]; then
      die "CLUSTER_PROVIDER=$CLUSTER_PROVIDER：不删除外部集群，请用 scripts/down.sh apps 并手动删除 acme / middleware / observability 命名空间"
    fi
    "$LC_DIR/scripts/cluster.sh" delete
    rm -f "$LC_DIR"/.image-tag-* "$LC_DIR/.kind-config.yaml"
    ;;
  *)
    die "用法：$0 [cluster | apps]"
    ;;
esac
