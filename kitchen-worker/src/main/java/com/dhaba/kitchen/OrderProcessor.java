package com.dhaba.kitchen;

import java.time.Duration;
import java.util.logging.Logger;

/**
 * Cooks one order: new -> cooking -> (wait) -> ready.
 * Plain Java on purpose, so it is easy to unit test.
 */
public class OrderProcessor {

    /** Lets tests skip the real waiting. */
    @FunctionalInterface
    public interface Pause {
        void sleep(Duration duration) throws InterruptedException;
    }

    private static final Logger LOG = Logger.getLogger(OrderProcessor.class.getName());

    private final OrderRepository orders;
    private final Duration cookingTime;
    private final Pause pause;

    public OrderProcessor(OrderRepository orders, Duration cookingTime, Pause pause) {
        this.orders = orders;
        this.cookingTime = cookingTime;
        this.pause = pause;
    }

    /**
     * @return true if the order was cooked to ready, false if it was skipped
     *         (unknown order or already taken, e.g. a duplicate job)
     */
    public boolean process(long orderId) throws InterruptedException {
        if (!orders.advance(orderId, OrderRepository.NEW, OrderRepository.COOKING)) {
            LOG.warning("Order " + orderId + " is not in status 'new', skipping");
            return false;
        }
        LOG.info("Order " + orderId + " is cooking");
        pause.sleep(cookingTime);
        if (!orders.advance(orderId, OrderRepository.COOKING, OrderRepository.READY)) {
            LOG.warning("Order " + orderId + " was not in status 'cooking' when finished");
            return false;
        }
        LOG.info("Order " + orderId + " is ready");
        return true;
    }
}
