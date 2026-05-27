package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lqhiyul/personality-type-test/internal/config"
	storagesqlite "github.com/lqhiyul/personality-type-test/internal/storage/sqlite"
)

func main() {
	cfg := config.FromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	db, err := storagesqlite.Open(context.Background(), cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Printf("migrations applied to %s\n", cfg.DatabasePath)
}
