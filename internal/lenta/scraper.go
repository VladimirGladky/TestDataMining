package lenta

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"
)

type Product struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	Price        float64 `json:"price"`
	PriceRegular float64 `json:"price_regular"`
	HasDiscount  bool    `json:"has_discount"`
	Unit         string  `json:"unit"`
	UnitQuantity float64 `json:"unit_quantity"`
	Rating       float64 `json:"rating"`
	RatingVotes  int     `json:"rating_votes"`
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
}

type CategoryResult struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Products []Product `json:"products"`
}

func (c *Client) ScrapeSelection(ctx context.Context, selectionID int64) (*CategoryResult, error) {
	first, err := c.FetchSelectionPage(ctx, selectionID, 0, c.cfg.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("fetch first page (selection %d): %w", selectionID, err)
	}

	categoryName := nameFromGroups(first.SelectionGroups)
	if categoryName == "" {
		categoryName = c.cfg.SelectionNames[selectionID]
	}
	result := &CategoryResult{
		ID:       selectionID,
		Name:     categoryName,
		Products: make([]Product, 0, first.Total),
	}
	appendProducts(result, first.Items, selectionID, categoryName)

	log.Printf("selection %d (%s): total=%d, page1=%d", selectionID, categoryName, first.Total, len(first.Items))

	for offset := c.cfg.PageLimit; offset < first.Total; offset += c.cfg.PageLimit {
		politeSleep(ctx)
		page, err := c.FetchSelectionPage(ctx, selectionID, offset, c.cfg.PageLimit)
		if err != nil {
			return nil, fmt.Errorf("fetch offset %d: %w", offset, err)
		}
		if len(page.Items) == 0 {
			break
		}
		appendProducts(result, page.Items, selectionID, categoryName)
		log.Printf("selection %d: fetched offset=%d (%d items, accumulated=%d/%d)", selectionID, offset, len(page.Items), len(result.Products), first.Total)
	}
	return result, nil
}

func appendProducts(out *CategoryResult, items []Item, categoryID int64, categoryName string) {
	for _, it := range items {
		out.Products = append(out.Products, Product{
			ID:           it.ID,
			Name:         it.Name,
			URL:          ProductURL(it.Slug, it.ID),
			Price:        kopecksToRubles(it.Prices.Price),
			PriceRegular: kopecksToRubles(it.Prices.PriceRegular),
			HasDiscount:  it.Prices.IsPromoactionPrice && it.Prices.Price < it.Prices.PriceRegular,
			Unit:         it.Units.SaleUnit,
			UnitQuantity: it.Units.SaleUnitQuantity,
			Rating:       it.Rating.Rate,
			RatingVotes:  it.Rating.Votes,
			CategoryID:   categoryID,
			CategoryName: categoryName,
		})
	}
}

func nameFromGroups(groups []SelectionGroup) string {
	if len(groups) == 0 {
		return ""
	}
	return groups[0].Name
}

func kopecksToRubles(k int64) float64 {
	return float64(k) / 100.0
}

func politeSleep(ctx context.Context) {
	d := time.Duration(800+rand.IntN(900)) * time.Millisecond
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
