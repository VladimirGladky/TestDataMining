package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"TestDataMining/internal/config"
	"TestDataMining/internal/lenta"
	"TestDataMining/internal/output"
)

func main() {
	envPath := flag.String("env", ".env", "path to .env file")
	outPath := flag.String("out", "out/products.json", "path to output JSON file")
	flag.Parse()

	cfg, err := config.Load(*envPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client, err := lenta.NewClient(cfg)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	results := make([]lenta.CategoryResult, 0, len(cfg.SelectionIDs))
	for _, id := range cfg.SelectionIDs {
		log.Printf("scraping selection %d via proxy...", id)
		res, err := client.ScrapeSelection(ctx, id)
		if err != nil {
			log.Fatalf("selection %d: %v", id, err)
		}
		log.Printf("selection %d done: %d products", id, len(res.Products))
		results = append(results, *res)
	}

	if err := output.WriteJSON(*outPath, results); err != nil {
		log.Fatalf("write output: %v", err)
	}

	log.Printf("done. wrote %s", *outPath)
}
