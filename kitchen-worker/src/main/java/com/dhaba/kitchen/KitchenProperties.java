package com.dhaba.kitchen;

import java.time.Duration;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "kitchen")
public record KitchenProperties(String queueKey, long cookingSeconds, long pollSeconds) {

    public Duration cookingTime() {
        return Duration.ofSeconds(cookingSeconds);
    }

    public Duration pollTimeout() {
        return Duration.ofSeconds(pollSeconds);
    }
}
