package com.dhaba.kitchen;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.junit.jupiter.api.Test;

class OrderProcessorTest {

    /** In-memory stand-in for the orders table. */
    static class FakeOrders implements OrderRepository {
        final Map<Long, String> status = new HashMap<>();
        final List<String> log = new ArrayList<>();

        @Override
        public boolean advance(long orderId, String expected, String next) {
            if (!expected.equals(status.get(orderId))) {
                return false;
            }
            status.put(orderId, next);
            log.add(orderId + ":" + expected + "->" + next);
            return true;
        }
    }

    @Test
    void cooksOrderFromNewToReady() throws Exception {
        FakeOrders orders = new FakeOrders();
        orders.status.put(7L, "new");
        List<Duration> pauses = new ArrayList<>();
        OrderProcessor processor = new OrderProcessor(orders, Duration.ofSeconds(8), pauses::add);

        assertTrue(processor.process(7));

        assertEquals("ready", orders.status.get(7L));
        assertEquals(List.of("7:new->cooking", "7:cooking->ready"), orders.log);
        assertEquals(List.of(Duration.ofSeconds(8)), pauses);
    }

    @Test
    void waitsWhileOrderIsCooking() throws Exception {
        FakeOrders orders = new FakeOrders();
        orders.status.put(1L, "new");
        OrderProcessor processor = new OrderProcessor(orders, Duration.ofSeconds(3), d -> {
            assertEquals("cooking", orders.status.get(1L), "must be cooking while waiting");
        });
        assertTrue(processor.process(1));
    }

    @Test
    void skipsUnknownOrder() throws Exception {
        FakeOrders orders = new FakeOrders();
        List<Duration> pauses = new ArrayList<>();
        OrderProcessor processor = new OrderProcessor(orders, Duration.ofSeconds(8), pauses::add);

        assertFalse(processor.process(99));
        assertTrue(pauses.isEmpty(), "no cooking time for an order that does not exist");
    }

    @Test
    void duplicateJobDoesNotCookTwice() throws Exception {
        FakeOrders orders = new FakeOrders();
        orders.status.put(5L, "new");
        OrderProcessor processor = new OrderProcessor(orders, Duration.ZERO, d -> { });

        assertTrue(processor.process(5));
        assertFalse(processor.process(5));
        assertEquals(2, orders.log.size());
    }

    @Test
    void skipsOrderAlreadyCooking() throws Exception {
        FakeOrders orders = new FakeOrders();
        orders.status.put(3L, "cooking");
        OrderProcessor processor = new OrderProcessor(orders, Duration.ZERO, d -> { });

        assertFalse(processor.process(3));
        assertEquals("cooking", orders.status.get(3L));
    }

    @Test
    void interruptionPropagatesAndLeavesOrderCooking() {
        FakeOrders orders = new FakeOrders();
        orders.status.put(2L, "new");
        OrderProcessor processor = new OrderProcessor(orders, Duration.ofSeconds(8), d -> {
            throw new InterruptedException("shutdown");
        });

        assertThrows(InterruptedException.class, () -> processor.process(2));
        assertEquals("cooking", orders.status.get(2L));
    }
}
