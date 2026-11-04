package com.acme.gateway.config;

import io.micrometer.observation.ObservationPredicate;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.server.reactive.observation.ServerRequestObservationContext;

@Configuration(proxyBeanMethods = false)
public class ObservationConfig {

    /** 不为管理端点（探针、Prometheus 抓取）生成链路，避免淹没业务链路。 */
    @Bean
    public ObservationPredicate skipActuatorObservations() {
        return (name, context) -> !(context instanceof ServerRequestObservationContext request
                && request.getCarrier().getPath().value().startsWith("/actuator"));
    }
}
