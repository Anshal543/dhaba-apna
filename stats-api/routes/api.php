<?php

use App\Http\Controllers\StatsController;
use Illuminate\Support\Facades\Route;

Route::get('/health', [StatsController::class, 'health']);
Route::get('/stats', [StatsController::class, 'show']);
