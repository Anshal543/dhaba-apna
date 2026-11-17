<?php

namespace Tests\Feature;

use App\Contracts\StatsSource;
use RuntimeException;
use Tests\TestCase;

class StatsApiTest extends TestCase
{
    public function test_health_is_ok(): void
    {
        $this->getJson('/api/health')
            ->assertOk()
            ->assertExactJson(['status' => 'ok']);
    }

    public function test_stats_returns_numbers_from_the_source(): void
    {
        $this->app->instance(StatsSource::class, new class implements StatsSource {
            public function ordersInMinutes(array $minutes): array
            {
                return [end($minutes) => 7];
            }

            public function topDishes(int $limit): array
            {
                return [['name' => 'Chicken Karahi', 'count' => 21]];
            }
        });

        $this->getJson('/api/stats')
            ->assertOk()
            ->assertJsonPath('orders_per_minute', 7)
            ->assertJsonCount(10, 'history')
            ->assertJsonPath('top_dishes.0.name', 'Chicken Karahi')
            ->assertJsonPath('top_dishes.0.count', 21);
    }

    public function test_stats_returns_503_when_redis_is_down(): void
    {
        $this->app->instance(StatsSource::class, new class implements StatsSource {
            public function ordersInMinutes(array $minutes): array
            {
                throw new RuntimeException('connection refused');
            }

            public function topDishes(int $limit): array
            {
                return [];
            }
        });

        $this->getJson('/api/stats')
            ->assertStatus(503)
            ->assertJsonPath('error', 'stats are unavailable');
    }
}
