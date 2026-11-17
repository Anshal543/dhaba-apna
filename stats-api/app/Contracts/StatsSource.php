<?php

namespace App\Contracts;

/**
 * Where the raw numbers come from. Redis in production, a fake in tests.
 */
interface StatsSource
{
    /**
     * Orders created in each given minute (minute = unix time / 60, integer division).
     *
     * @param  int[]  $minutes
     * @return array<int,int> minute => number of orders (0 when nothing was recorded)
     */
    public function ordersInMinutes(array $minutes): array;

    /**
     * Most ordered dishes, highest first.
     *
     * @return array<int,array{name:string,count:int}>
     */
    public function topDishes(int $limit): array;
}
