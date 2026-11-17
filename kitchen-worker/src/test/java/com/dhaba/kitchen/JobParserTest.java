package com.dhaba.kitchen;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.NullAndEmptySource;
import org.junit.jupiter.params.provider.ValueSource;

class JobParserTest {

    private final JobParser parser = new JobParser();

    @Test
    void readsOrderId() {
        assertEquals(42L, parser.parseOrderId("{\"order_id\":42}"));
    }

    @Test
    void ignoresExtraFieldsAndWhitespace() {
        assertEquals(7L, parser.parseOrderId(" { \"order_id\" : 7, \"extra\": true } "));
    }

    @Test
    void readsLargeIds() {
        assertEquals(9_000_000_000L, parser.parseOrderId("{\"order_id\":9000000000}"));
    }

    @ParameterizedTest
    @NullAndEmptySource
    @ValueSource(strings = {
            "   ",
            "not json",
            "{}",
            "[]",
            "42",
            "{\"order_id\":\"42\"}",
            "{\"order_id\":4.5}",
            "{\"order_id\":0}",
            "{\"order_id\":-1}",
            "{\"order_id\":null}",
            "{\"id\":42}"
    })
    void rejectsBadJobs(String message) {
        assertThrows(IllegalArgumentException.class, () -> parser.parseOrderId(message));
    }
}
