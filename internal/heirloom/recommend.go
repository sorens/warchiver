package heirloom

// Recommendation is a suggested purchase or upgrade action for a single heirloom.
type Recommendation struct {
	HeirloomID int
	Name       string
	Action     string // "purchase" or "upgrade"
	Currency   Currency
	Amount     int
	Reason     string
}

// Recommend returns purchase/upgrade recommendations ordered by the given currency
// priority list. Lower index in currencyRanks means higher priority.
//
// Stub: returns nil until ranking logic is implemented.
func Recommend(report *Report, currencyRanks []Currency) []Recommendation {
	return nil
}
