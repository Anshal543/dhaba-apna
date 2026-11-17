<?php

namespace App\Services;

use App\Contracts\StatsSource;
use Closure;

class StatsService
{
    public const HISTORY_MINUTES = 10;
    public const TOP_DISHES = 5;

    /** @var Closure():int */
    private Closure $clock;

    /**
     * @param  Closure():int|null  $clock  returns the current unix time; injectable for tests
     */
    public function __construct(private StatsSource $source, ?Closure $clock = null)
    {
        $this->clock = $clock ?? fn (): int => time();
    }

    /**
     * @return array{
     *   orders_per_minute:int,
     *   history:array<int,array{minute:string,orders:int}>,
     *   top_dishes:array<int,array{name:string,count:int}>
     * }
     */
    public function snapshot(): array
    {
        $currentMinute = intdiv(($this->clock)(), 60);

        // Only complete minutes: the current one is still filling up.
        $minutes = range($currentMinute - self::HISTORY_MINUTES, $currentMinute - 1);
        $counts = $this->source->ordersInMinutes($minutes);

        $history = [];
        foreach ($minutes as $minute) {
            $history[] = [
                'minute' => gmdate('Y-m-d\TH:i:s\Z', $minute * 60),
                'orders' => (int) ($counts[$minute] ?? 0),
            ];
        }

        return [
            'orders_per_minute' => $history[count($history) - 1]['orders'],
            'history' => $history,
            'top_dishes' => $this->source->topDishes(self::TOP_DISHES),
        ];
    }
}
