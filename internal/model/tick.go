package model

type Tick struct {
	Market    string
	Token     string
	Price     float64
	Volume    float64
	Timestamp int64
}
