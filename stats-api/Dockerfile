FROM php:8.4-cli-alpine

RUN apk add --no-cache git unzip \
    && addgroup -S app && adduser -S -G app app

COPY --from=composer:2 /usr/bin/composer /usr/local/bin/composer

WORKDIR /app

COPY composer.json composer.lock ./
RUN composer install --no-dev --prefer-dist --no-interaction --no-autoloader --no-scripts

COPY . .
RUN composer dump-autoload --optimize --no-dev \
    && chown -R app:app /app/storage /app/bootstrap/cache

USER app

EXPOSE 8001

CMD ["php", "artisan", "serve", "--host", "0.0.0.0", "--port", "8001"]
