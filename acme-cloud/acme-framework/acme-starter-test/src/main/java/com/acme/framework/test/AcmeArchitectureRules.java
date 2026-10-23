package com.acme.framework.test;

import static com.tngtech.archunit.lang.syntax.ArchRuleDefinition.noClasses;
import static com.tngtech.archunit.library.Architectures.layeredArchitecture;

import com.tngtech.archunit.lang.ArchRule;

/**
 * 团队分层规范的可执行版本（每个服务在 ArchitectureTest 中引用）：
 * <pre>
 * interfaces（Controller / 消息消费者）
 *     ↓
 * application（应用服务：编排、事务、发事件）
 *     ↓
 * domain（领域模型 + 仓储接口，纯 Java）
 *     ↑
 * infrastructure（仓储实现、Mapper、Feign 适配、外部系统）
 * </pre>
 */
public final class AcmeArchitectureRules {

    private AcmeArchitectureRules() {
    }

    public static ArchRule layers() {
        return layeredArchitecture().consideringOnlyDependenciesInLayers()
                .layer("Interfaces").definedBy("..interfaces..")
                .layer("Application").definedBy("..application..")
                .layer("Domain").definedBy("..domain..")
                .layer("Infrastructure").definedBy("..infrastructure..")
                .whereLayer("Interfaces").mayNotBeAccessedByAnyLayer()
                .whereLayer("Application").mayOnlyBeAccessedByLayers("Interfaces")
                .whereLayer("Domain").mayOnlyBeAccessedByLayers("Interfaces", "Application", "Infrastructure")
                .whereLayer("Infrastructure").mayOnlyBeAccessedByLayers("Application")
                .allowEmptyShould(true);
    }

    /** 领域层不依赖 Spring Web、MyBatis、Feign 等技术框架。 */
    public static ArchRule domainIsFrameworkFree() {
        return noClasses().that().resideInAPackage("..domain..")
                .should().dependOnClassesThat().resideInAnyPackage(
                        "org.springframework.web..", "com.baomidou..", "org.apache.ibatis..", "feign..",
                        "org.springframework.cloud..")
                .allowEmptyShould(true);
    }

    /** Controller 不能直接访问 Mapper / DO。 */
    public static ArchRule controllersDoNotTouchPersistence() {
        return noClasses().that().resideInAPackage("..interfaces..")
                .should().dependOnClassesThat().resideInAPackage("..infrastructure.persistence..")
                .allowEmptyShould(true);
    }
}
