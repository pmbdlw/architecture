package com.acme.order;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.cloud.openfeign.EnableFeignClients;

import com.acme.user.api.UserClient;

/**
 * 显式列出用到的 Feign 客户端，而不是扫描整个包，依赖关系一目了然。
 */
@SpringBootApplication
@EnableFeignClients(clients = UserClient.class)
public class OrderApplication {

    public static void main(String[] args) {
        SpringApplication.run(OrderApplication.class, args);
    }
}
