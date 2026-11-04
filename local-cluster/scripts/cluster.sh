#!/usr/bin/env bash
# 创建 / 删除本地 kind 集群：1 个控制面 + N 个工作节点（默认 3 个，分别标记为 zone-a/b/c 三个“可用区”）。
#   scripts/cluster.sh create
#   scripts/cluster.sh delete
source "$(dirname "$0")/lib.sh"

render_config() {
  local zones=(zone-a zone-b zone-c)
  cat <<EOF
# 由 scripts/cluster.sh 生成。模拟生产集群的形态：控制面与工作节点分离，工作节点分布在 3 个可用区。
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: $CLUSTER_NAME
nodes:
  - role: control-plane
    # 宿主机端口 → NodePort：只绑定 127.0.0.1，不暴露到局域网
    extraPortMappings:
      - { containerPort: 30080, hostPort: 30080, listenAddress: 127.0.0.1 }  # API 网关
      - { containerPort: 30900, hostPort: 30900, listenAddress: 127.0.0.1 }  # 认证中心
      - { containerPort: 30300, hostPort: 30300, listenAddress: 127.0.0.1 }  # Grafana
      - { containerPort: 30686, hostPort: 30686, listenAddress: 127.0.0.1 }  # Jaeger UI
      - { containerPort: 30090, hostPort: 30090, listenAddress: 127.0.0.1 }  # Prometheus
      - { containerPort: 30848, hostPort: 30848, listenAddress: 127.0.0.1 }  # Nacos 控制台
      - { containerPort: 30672, hostPort: 30672, listenAddress: 127.0.0.1 }  # RabbitMQ 管理台（MQ=rabbitmq）
EOF
  for ((i = 0; i < KIND_WORKERS; i++)); do
    cat <<EOF
  - role: worker
    labels:
      topology.kubernetes.io/zone: ${zones[$((i % 3))]}
EOF
  done
  cat <<'EOF'
# 兼容在容器或受限沙箱（部分 CI、云开发机）中运行 kind：不允许调低 oom_score_adj 时 Pod 无法启动。
# 在普通开发机上这几项没有可感知的影响。
containerdConfigPatches:
  - |-
    [plugins."io.containerd.cri.v1.runtime"]
      restrict_oom_score_adj = true
kubeadmConfigPatches:
  - |
    kind: KubeletConfiguration
    oomScoreAdj: 0
  - |
    kind: KubeProxyConfiguration
    oomScoreAdj: 0
EOF
}

case "${1:-}" in
  create)
    need kind kubectl docker
    if kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
      ok "kind 集群 $CLUSTER_NAME 已存在"
    else
      node_image=$KIND_NODE_IMAGE
      if [[ -n "$IMAGE_MIRROR" ]]; then node_image="$IMAGE_MIRROR/$KIND_NODE_IMAGE"; fi
      log "创建 kind 集群 $CLUSTER_NAME（1 控制面 + $KIND_WORKERS 工作节点，节点镜像 $node_image）"
      render_config > "$LC_DIR/.kind-config.yaml"
      kind create cluster --config "$LC_DIR/.kind-config.yaml" --image "$node_image" --wait 120s
    fi
    k get nodes -L topology.kubernetes.io/zone
    ;;
  delete)
    kind delete cluster --name "$CLUSTER_NAME"
    ;;
  config)
    render_config
    ;;
  *)
    echo "用法：$0 create | delete | config" >&2
    exit 1
    ;;
esac
