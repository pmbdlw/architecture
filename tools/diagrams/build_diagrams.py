"""生成 docs/images/*.svg：python3 tools/diagrams/build_diagrams.py"""
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

import arch_diagrams  # noqa: E402
import env_cloud_diagrams  # noqa: E402
import springcloud_diagrams  # noqa: E402
import go_diagrams  # noqa: E402

ROOT = os.path.join(os.path.dirname(__file__), "..", "..")
OUT = os.path.join(ROOT, "docs", "images")
OUT_GO = os.path.join(ROOT, "docs-go", "images")

DIAGRAMS = {
    "arch-selfhosted.svg": arch_diagrams.selfhosted,
    "arch-aliyun.svg": arch_diagrams.aliyun,
    "environments.svg": env_cloud_diagrams.environments,
    "arch-aws-azure.svg": env_cloud_diagrams.aws_azure,
    "cloud-mapping.svg": env_cloud_diagrams.mapping,
    "spring-cloud-landscape.svg": springcloud_diagrams.landscape,
    "service-layers.svg": springcloud_diagrams.layers,
    "code-modules.svg": springcloud_diagrams.modules,
}

# Go 技术栈版本：运行架构类的图拓扑相同、只换组件标注；组件分布 / 分层 / 模块图为 Go 专属
DIAGRAMS_GO = {
    "arch-selfhosted.svg": lambda: arch_diagrams.selfhosted("go"),
    "arch-aliyun.svg": lambda: arch_diagrams.aliyun("go"),
    "environments.svg": lambda: env_cloud_diagrams.environments("go"),
    "arch-aws-azure.svg": lambda: env_cloud_diagrams.aws_azure("go"),
    "cloud-mapping.svg": lambda: env_cloud_diagrams.mapping("go"),
    "go-landscape.svg": go_diagrams.landscape,
    "service-layers.svg": go_diagrams.layers,
    "code-modules.svg": go_diagrams.modules,
}

if __name__ == "__main__":
    only = set(sys.argv[1:])
    for out, diagrams in ((OUT, DIAGRAMS), (OUT_GO, DIAGRAMS_GO)):
        os.makedirs(out, exist_ok=True)
        for name, fn in diagrams.items():
            if only and name not in only:
                continue
            fn().save(os.path.join(out, name))
            print("generated", os.path.relpath(os.path.join(out, name), ROOT))
