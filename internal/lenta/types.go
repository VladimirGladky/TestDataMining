package lenta

type SelectionRequest struct {
	SelectionID      int64   `json:"selectionId"`
	SelectionGroupID *int64  `json:"selectionGroupId"`
	Filters          Filters `json:"filters"`
	Sort             Sort    `json:"sort"`
	Limit            int     `json:"limit"`
	Offset           int     `json:"offset"`
}

type Filters struct {
	Checkbox      []any `json:"checkbox"`
	Multicheckbox []any `json:"multicheckbox"`
	Range         []any `json:"range"`
}

type Sort struct {
	Type  string `json:"type"`
	Order string `json:"order"`
}

type SelectionResponse struct {
	Items            []Item           `json:"items"`
	SelectionGroups  []SelectionGroup `json:"selectionGroups"`
	Total            int              `json:"total"`
	RecommendationID string           `json:"recommendationId"`
}

type SelectionGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type Item struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Prices Prices `json:"prices"`
	Units  Units  `json:"units"`
	Rating Rating `json:"rating"`
	Count  int    `json:"count"`
}

type Prices struct {
	Cost               int64 `json:"cost"`
	CostRegular        int64 `json:"costRegular"`
	Price              int64 `json:"price"`
	PriceRegular       int64 `json:"priceRegular"`
	IsLoyaltyCardPrice bool  `json:"isLoyaltyCardPrice"`
	IsPromoactionPrice bool  `json:"isPromoactionPrice"`
	IsQuantPrice       bool  `json:"isQuantPrice"`
}

type Units struct {
	ItemUnit         string  `json:"itemUnit"`
	ItemUnitQuantity float64 `json:"itemUnitQuantity"`
	SaleUnit         string  `json:"saleUnit"`
	SaleUnitQuantity float64 `json:"saleUnitQuantity"`
}

type Rating struct {
	Rate  float64 `json:"rate"`
	Votes int     `json:"votes"`
}
