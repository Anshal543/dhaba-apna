package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"dhaba/orders-api/internal/api"
	"dhaba/orders-api/internal/redisq"
	"dhaba/orders-api/internal/store"
)

func main() {
	db, err := sql.Open("postgres", postgresDSN())
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	pg := store.NewPostgres(db)
	rc := redisq.NewClient(
		net.JoinHostPort(env("REDIS_HOST", "localhost"), env("REDIS_PORT", "6379")),
		os.Getenv("REDIS_PASSWORD"),
	)
	pub := redisq.NewPublisher(rc)

	// Containers often start before Postgres is ready; wait a bit instead of crashing.
	waitFor("postgres", pg.Ping, 30*time.Second)

	pod, _ := os.Hostname()
	srv := &api.Server{
		Store:      pg,
		Queue:      pub,
		Stats:      pub,
		Version:    env("APP_VERSION", "dev"),
		Pod:        env("HOSTNAME", pod),
		CORSOrigin: env("CORS_ORIGIN", "*"),
		Checks: map[string]func(context.Context) error{
			"postgres": pg.Ping,
			"redis":    pub.Ping,
		},
	}

	httpSrv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("orders-api %s listening on %s", srv.Version, httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func postgresDSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(env("DB_USER", "dhaba"), env("DB_PASSWORD", "dhaba_local_dev")),
		Host:     net.JoinHostPort(env("DB_HOST", "localhost"), env("DB_PORT", "5432")),
		Path:     env("DB_NAME", "dhaba"),
		RawQuery: "sslmode=" + env("DB_SSLMODE", "disable"),
	}
	return u.String()
}



func waitFor(name string, ping func(context.Context) error, total time.Duration) {
	deadline := time.Now().Add(total)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := ping(ctx)
		cancel()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			log.Printf("%s not reachable after %s (%v); starting anyway, /ready will report 503", name, total, err)
			return
		}
		log.Printf("waiting for %s: %v", name, err)
		time.Sleep(2 * time.Second)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
