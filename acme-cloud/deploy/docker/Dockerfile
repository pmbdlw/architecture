# syntax=docker/dockerfile:1
# =====================================================================
# 所有 Spring Boot 服务共用的镜像构建文件（在 acme-cloud 根目录执行）：
#   mvn -B package -DskipTests
#   docker build -f deploy/docker/Dockerfile \
#     --build-arg MODULE=acme-services/order-service/order-server \
#     -t registry.acme.com/acme/order-service:$(git rev-parse --short HEAD) .
#
# 分层：依赖层很少变化，可被镜像仓库和节点缓存复用；每次发布通常只推送几百 KB 的 application 层。
# 基础镜像可替换为企业镜像仓库中的同版本镜像（自建 Harbor / 阿里云 ACR / Amazon ECR / Azure ACR 的代理缓存）。
# =====================================================================
ARG JRE_IMAGE=eclipse-temurin:21-jre-noble

FROM ${JRE_IMAGE} AS extract
ARG MODULE
WORKDIR /work
COPY ${MODULE}/target/app.jar app.jar
RUN java -Djarmode=tools -jar app.jar extract --layers --launcher --destination extracted

FROM ${JRE_IMAGE}
RUN groupadd --system --gid 10001 app \
    && useradd --system --uid 10001 --gid app --no-create-home --shell /usr/sbin/nologin app
WORKDIR /app
COPY --from=extract /work/extracted/dependencies/ ./
COPY --from=extract /work/extracted/spring-boot-loader/ ./
COPY --from=extract /work/extracted/snapshot-dependencies/ ./
COPY --from=extract /work/extracted/application/ ./
USER 10001
EXPOSE 8080 8081
# JVM 参数通过 JAVA_TOOL_OPTIONS 环境变量注入（见 Helm values 的 jvm.options）
ENTRYPOINT ["java", "org.springframework.boot.loader.launch.JarLauncher"]
