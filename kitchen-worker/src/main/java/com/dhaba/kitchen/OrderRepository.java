package com.dhaba.kitchen;

public interface OrderRepository {

    String NEW = "new";
    String COOKING = "cooking";
    String READY = "ready";

    /**
     * Moves an order from one status to the next, but only if it is still in
     * the expected status. Returns false when nothing was updated (unknown
     * order, or another worker already took it).
     */
    boolean advance(long orderId, String expectedStatus, String nextStatus);
}
