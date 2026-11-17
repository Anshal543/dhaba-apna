package com.dhaba.kitchen;

import java.time.Duration;
import java.util.logging.Level;
import java.util.logging.Logger;

import org.springframework.context.SmartLifecycle;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;

/**
 * Background loop: wait for a job on the Redis list, then cook that order.
 * orders-api does LPUSH, so RPOP here gives first-in-first-out.
 */
@Component
public class QueueListener implements SmartLifecycle {

    private static final Logger LOG = Logger.getLogger(QueueListener.class.getName());
    private static final Duration RETRY_DELAY = Duration.ofSeconds(2);

    private final StringRedisTemplate redis;
    private final KitchenProperties props;
    private final OrderRepository orders;
    private final JobParser parser = new JobParser();

    private volatile boolean running;
    private Thread thread;

    public QueueListener(StringRedisTemplate redis, KitchenProperties props, OrderRepository orders) {
        this.redis = redis;
        this.props = props;
        this.orders = orders;
    }

    @Override
    public void start() {
        running = true;
        thread = new Thread(this::loop, "kitchen-queue");
        thread.start();
        LOG.info("Listening on queue '" + props.queueKey() + "'");
    }

    @Override
    public void stop() {
        running = false;
        if (thread != null) {
            thread.interrupt();
        }
    }

    @Override
    public boolean isRunning() {
        return running;
    }

    private void loop() {
        OrderProcessor processor = new OrderProcessor(orders, props.cookingTime(), d -> Thread.sleep(d.toMillis()));
        while (running) {
            try {
                String message = redis.opsForList().rightPop(props.queueKey(), props.pollTimeout());
                if (message == null) {
                    continue; // timed out, poll again
                }
                handle(processor, message);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return;
            } catch (Exception e) {
                // Redis or Postgres hiccup: log, back off, keep running.
                if (!running) {
                    return;
                }
                LOG.log(Level.WARNING, "Queue loop error, retrying: " + e.getMessage());
                try {
                    Thread.sleep(RETRY_DELAY.toMillis());
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    return;
                }
            }
        }
    }

    private void handle(OrderProcessor processor, String message) throws InterruptedException {
        long orderId;
        try {
            orderId = parser.parseOrderId(message);
        } catch (IllegalArgumentException e) {
            LOG.warning("Dropping bad job: " + e.getMessage());
            return;
        }
        processor.process(orderId);
    }
}
