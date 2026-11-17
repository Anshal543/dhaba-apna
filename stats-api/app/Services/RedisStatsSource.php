<?php

namespace App\Services;

use App\Contracts\StatsSource;
use Illuminate\Support\Facades\Redis;

/**
 * Reads the keys orders-api writes (see docs/CONTRACT.md):
 *   stats:orders:<minute>  counter
 *   stats:dishes           sorted set, score = quantity ordered
 */
class RedisStatsSource implements StatsSource
{
    public function ordersInMinutes(array $minutes): array
    {
        if ($minutes === []) {
            return [];
        }

        $keys = array_map(fn (int $m): string => "stats:orders:{$m}", array_values($minutes));
        $values = Redis::mget($keys);

        $result = [];
        foreach (array_values($minutes) as $i => $minute) {
            $result[$minute] = (int) ($values[$i] ?? 0);
        }

        return $result;
    }

    public function topDishes(int $limit): array
    {
        if ($limit < 1) {
            return [];
        }

        $rows = Redis::zrevrange('stats:dishes', 0, $limit - 1, ['withscores' => true]);

        $dishes = [];
        foreach ($rows as $name => $score) {
            $dishes[] = ['name' => (string) $name, 'count' => (int) $score];
        }

        return $dishes;
    }
}
