package model

type Product struct {
	SKU   int64
	Name  string
	Price uint32
}

type CartItem struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type Cart struct {
	Items      []CartItem `json:"items"`
	TotalPrice uint32     `json:"total_price"`
}
