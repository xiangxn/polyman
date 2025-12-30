package model

type SubscribeUserMessage struct {
	Type    string   `json:"type"`
	Markets []string `json:"markets"`
	Auth    any      `json:"auth"`
}

type ClobAuth struct {
	ApiKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

type WSOrder struct {
	AssetId         string   `json:"asset_id"`
	AssociateTrades []string `json:"associate_trades"`
	EventType       string   `json:"event_type"`
	Id              string   `json:"id"`
	Market          string   `json:"market"`
	OrderOwner      string   `json:"order_owner"`
	OriginalSize    string   `json:"original_size"`
	Outcome         string   `json:"outcome"`
	Owner           string   `json:"owner"`
	Price           float64  `json:"price"`
	Side            string   `json:"side"`
	SizeMatched     float64  `json:"size_matched"`
	Timestamp       int64    `json:"timestamp"`
	Type            string   `json:"type"`
}

type WSMakerOrder struct {
	AssetId       string  `json:"asset_id"`
	MatchedAmount float64 `json:"matched_amount"`
	OrderId       string  `json:"order_id"`
	Outcome       string  `json:"outcome"`
	Owner         string  `json:"owner"`
	Side          string  `json:"side"`
	Price         float64 `json:"price"`
	FeeRateBps    float64 `json:"fee_rate_bps"`
}

type WSTrade struct {
	AssetId      string         `json:"asset_id"`
	EventType    string         `json:"event_type"`
	Id           string         `json:"id"`
	LastUpdate   int64          `json:"last_update"`
	MakerOrders  []WSMakerOrder `json:"maker_orders"`
	Market       string         `json:"market"`
	Matchtime    int64          `json:"matchtime"`
	Outcome      string         `json:"outcome"`
	Owner        string         `json:"owner"`
	Price        float64        `json:"price"`
	Side         string         `json:"side"`
	Size         float64        `json:"size"`
	FeeRateBps   float64        `json:"fee_rate_bps"`
	Status       string         `json:"status"`
	TakerOrderId string         `json:"taker_order_id"`
	Timestamp    int64          `json:"timestamp"`
	TradeOwner   string         `json:"trade_owner"`
	Type         string         `json:"type"`
}
