<?php

namespace App\Http\Controllers;

use App\Services\StatsService;
use Illuminate\Http\JsonResponse;
use Throwable;

class StatsController extends Controller
{
    public function health(): JsonResponse
    {
        return response()->json(['status' => 'ok']);
    }

    public function show(StatsService $stats): JsonResponse
    {
        try {
            return response()->json($stats->snapshot());
        } catch (Throwable $e) {
            report($e);

            return response()->json(['error' => 'stats are unavailable'], 503);
        }
    }
}
