<?php

// stats-api has no SQL database: it only reads counters from Redis.
return [
    'default' => 'null',

    'connections' => [
        'null' => ['driver' => 'sqlite', 'database' => ':memory:'],
    ],

    'migrations' => [
        'table' => 'migrations',
    ],

    'redis' => [
        'client' => env('REDIS_CLIENT', 'predis'),

        'options' => [
            // Keys are shared with orders-api, so no Laravel key prefix.
            'prefix' => '',
        ],

        'default' => [
            'host' => env('REDIS_HOST', '127.0.0.1'),
            'port' => env('REDIS_PORT', '6379'),
            'password' => env('REDIS_PASSWORD') ?: null,
            'database' => env('REDIS_DB', '0'),
        ],
    ],
];
