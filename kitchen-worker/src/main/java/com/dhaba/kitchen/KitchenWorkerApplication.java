package com.dhaba.kitchen;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class KitchenWorkerApplication {

    public static void main(String[] args) {
        SpringApplication.run(KitchenWorkerApplication.class, args);
    }
}
