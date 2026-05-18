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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("bootstrap: launching headless Chrome through %s...", cfg.ProxyURL)
	session, err := lenta.Bootstrap(ctx, lenta.BootstrapOptions{
		ProxyURL:  cfg.ProxyURL,
		UserAgent: cfg.UserAgent,
		Wait:      cfg.BootstrapWait,
	})
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	log.Printf("bootstrap: got sessionToken=%s..., deviceID=%s..., %d cookies",
		short(session.SessionToken), short(session.DeviceID), countSemis(session.Cookie))

	client, err := lenta.NewClient(cfg, session)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	log.Printf("locating store matching %q...", cfg.StoreQuery)
	store, err := client.FindStore(ctx, cfg.StoreQuery)
	if err != nil {
		log.Fatalf("find store: %v", err)
	}
	log.Printf("found store id=%d alias=%s title=%s (%s)", store.ID, store.Alias, store.Title, store.AddressFull)

	if err := client.SelectPickupStore(ctx, store.ID); err != nil {
		log.Fatalf("select store: %v", err)
	}
	log.Printf("store %d (%s) selected", store.ID, store.Alias)

	results := make([]lenta.CategoryResult, 0, len(cfg.SelectionIDs))
	for _, id := range cfg.SelectionIDs {
		log.Printf("scraping selection %d...", id)
		res, err := client.ScrapeSelection(ctx, id)
		if err != nil {
			log.Printf("selection %d FAILED: %v (keeping what we have so far)", id, err)
			continue
		}
		log.Printf("selection %d done: %d products", id, len(res.Products))
		results = append(results, *res)
	}

	if len(results) == 0 {
		log.Fatal("no categories collected")
	}

	if err := output.WriteJSON(*outPath, results); err != nil {
		log.Fatalf("write output: %v", err)
	}
	log.Printf("done. wrote %s", *outPath)
}

func short(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

func countSemis(s string) int {
	n := 1
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			n++
		}
	}
	if s == "" {
		return 0
	}
	return n
}
