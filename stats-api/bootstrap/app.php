<?php

use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;

$app = Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        api: __DIR__.'/../routes/api.php',
        health: '/up',
    )
    ->withMiddleware(function (Middleware $middleware) {
        //
    })
    ->withExceptions(function (Exceptions $exceptions) {
        //
    })->create();

// The repo has one env file. Laravel otherwise looks for stats-api/.env.
$repoRoot = dirname(__DIR__, 2);
if (is_file($repoRoot.'/.env')) {
    $app->useEnvironmentPath($repoRoot);
}

return $app;
