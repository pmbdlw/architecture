"""生成 docs/images/*.svg：python3 tools/diagrams/build_diagrams.py"""
import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

import arch_diagrams  # noqa: E402
import env_cloud_diagrams  # noqa: E402
import springcloud_diagrams  # noqa: E402

OUT = os.path.join(os.path.dirname(__file__), "..", "..", "docs", "images")

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

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    only = set(sys.argv[1:])
    for name, fn in DIAGRAMS.items():
        if only and name not in only:
            continue
        fn().save(os.path.join(OUT, name))
        print("generated", name)
