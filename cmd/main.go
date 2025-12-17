package main

import (
	"context"
	"interview-task-worker-pool/internal/config"
	router "interview-task-worker-pool/internal/http"
	"interview-task-worker-pool/internal/http/handlers"
	"interview-task-worker-pool/internal/service"
	"interview-task-worker-pool/internal/store/memory"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.New()

	store := memory.New()

	service, err := service.New(store)
	if err != nil {
		log.Fatalf("service initiation failed: %v", err)
	}

	handler := handlers.New(service)

	router := router.New(handler)

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %s\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	<-stop
	log.Printf("shut down signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	log.Printf("shut down gracefully")
}
