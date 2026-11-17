<?php

namespace Tests\Unit;

use App\Contracts\StatsSource;
use App\Services\StatsService;
use PHPUnit\Framework\TestCase;

class StatsServiceTest extends TestCase
{
    // 2026-11-17 12:00:30 UTC -> current minute is the one starting 12:00:00
    private const NOW = 1794916830;

    private function source(array $counts = [], array $dishes = []): StatsSource
    {
        return new class($counts, $dishes) implements StatsSource {
            public array $askedMinutes = [];
            public int $askedLimit = 0;

            public function __construct(private array $counts, private array $dishes)
            {
            }

            public function ordersInMinutes(array $minutes): array
            {
                $this->askedMinutes = $minutes;
                $out = [];
                foreach ($minutes as $m) {
                    if (isset($this->counts[$m])) {
                        $out[$m] = $this->counts[$m];
                    }
                }

                return $out;
            }

            public function topDishes(int $limit): array
            {
                $this->askedLimit = $limit;

                return array_slice($this->dishes, 0, $limit);
            }
        };
    }

    public function test_asks_for_the_last_ten_complete_minutes(): void
    {
        $source = $this->source();
        (new StatsService($source, fn () => self::NOW))->snapshot();

        $current = intdiv(self::NOW, 60);
        $this->assertSame(range($current - 10, $current - 1), $source->askedMinutes);
        $this->assertNotContains($current, $source->askedMinutes, 'current minute is incomplete');
    }

    public function test_orders_per_minute_is_the_last_complete_minute(): void
    {
        $current = intdiv(self::NOW, 60);
        $source = $this->source([$current - 1 => 12, $current => 99, $current - 2 => 4]);

        $snapshot = (new StatsService($source, fn () => self::NOW))->snapshot();

        $this->assertSame(12, $snapshot['orders_per_minute']);
    }

    public function test_history_is_oldest_first_and_fills_missing_minutes_with_zero(): void
    {
        $current = intdiv(self::NOW, 60);
        $source = $this->source([$current - 10 => 3, $current - 1 => 5]);

        $history = (new StatsService($source, fn () => self::NOW))->snapshot()['history'];

        $this->assertCount(10, $history);
        $this->assertSame(3, $history[0]['orders']);
        $this->assertSame(5, $history[9]['orders']);
        $this->assertSame(0, $history[4]['orders']);
        $this->assertSame(
            gmdate('Y-m-d\TH:i:s\Z', ($current - 10) * 60),
            $history[0]['minute']
        );
        $this->assertSame('2026-11-17T11:59:00Z', $history[9]['minute']);
    }

    public function test_no_activity_gives_zeroes_and_no_dishes(): void
    {
        $snapshot = (new StatsService($this->source(), fn () => self::NOW))->snapshot();

        $this->assertSame(0, $snapshot['orders_per_minute']);
        $this->assertSame([], $snapshot['top_dishes']);
        $this->assertSame(array_fill(0, 10, 0), array_column($snapshot['history'], 'orders'));
    }

    public function test_top_dishes_come_from_the_source_limited_to_five(): void
    {
        $dishes = [];
        foreach (['Karahi', 'Naan', 'Biryani', 'Dal', 'Raita', 'Lassi', 'Kheer'] as $i => $name) {
            $dishes[] = ['name' => $name, 'count' => 100 - $i];
        }
        $source = $this->source([], $dishes);

        $snapshot = (new StatsService($source, fn () => self::NOW))->snapshot();

        $this->assertSame(5, $source->askedLimit);
        $this->assertCount(5, $snapshot['top_dishes']);
        $this->assertSame(['name' => 'Karahi', 'count' => 100], $snapshot['top_dishes'][0]);
    }

    public function test_minute_boundary_uses_integer_division(): void
    {
        $boundary = 60 * 30000000; // exactly on a minute
        $source = $this->source();
        (new StatsService($source, fn () => $boundary))->snapshot();

        $this->assertSame(30000000 - 1, end($source->askedMinutes));
    }
}
