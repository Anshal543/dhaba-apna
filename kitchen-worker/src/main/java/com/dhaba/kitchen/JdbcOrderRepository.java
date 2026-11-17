package com.dhaba.kitchen;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Repository;

@Repository
public class JdbcOrderRepository implements OrderRepository {

    private final JdbcTemplate jdbc;

    public JdbcOrderRepository(JdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    @Override
    public boolean advance(long orderId, String expectedStatus, String nextStatus) {
        int rows = jdbc.update(
                "UPDATE orders SET status = ?, updated_at = now() WHERE id = ? AND status = ?",
                nextStatus, orderId, expectedStatus);
        return rows == 1;
    }
}
