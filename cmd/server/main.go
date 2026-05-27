package main

import (
	"context"
	"log"

	"github.com/lqhiyul/personality-type-test/internal/app"
	"github.com/lqhiyul/personality-type-test/internal/config"
)

func main() {
	if err := app.Run(context.Background(), config.FromEnv()); err != nil {
		log.Fatal(err)
	}
}
