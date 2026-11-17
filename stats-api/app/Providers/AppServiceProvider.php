<?php

namespace App\Providers;

use App\Contracts\StatsSource;
use App\Services\RedisStatsSource;
use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->app->bind(StatsSource::class, RedisStatsSource::class);
    }

    public function boot(): void
    {
        //
    }
}
