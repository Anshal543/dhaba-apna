package com.dhaba.kitchen;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

/** Reads queue messages of the form {"order_id": 42}. */
public class JobParser {

    private final ObjectMapper mapper = new ObjectMapper();

    /** @throws IllegalArgumentException if the message is not a valid job */
    public long parseOrderId(String message) {
        if (message == null || message.isBlank()) {
            throw new IllegalArgumentException("empty job");
        }
        JsonNode root;
        try {
            root = mapper.readTree(message);
        } catch (JsonProcessingException e) {
            throw new IllegalArgumentException("job is not valid JSON: " + message, e);
        }
        JsonNode id = root == null ? null : root.get("order_id");
        if (id == null || !id.isIntegralNumber() || id.asLong() < 1) {
            throw new IllegalArgumentException("job has no valid order_id: " + message);
        }
        return id.asLong();
    }
}
