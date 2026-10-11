package heirloom

// Currency identifies a purchasable/spendable currency type.
type Currency string

const (
	CurrencyGold                      Currency = "gold"
	CurrencyMarkOfHonor               Currency = "mark_of_honor"
	CurrencyDarkmoonPrizeTicket       Currency = "darkmoon_prize_ticket"
	CurrencyChampionsSeal             Currency = "champions_seal"
	CurrencyHonourboundServiceMedal   Currency = "honorbound_service_medal"
	CurrencySeventhLegionServiceMedal Currency = "seventh_legion_service_medal"
	CurrencyTimewargedBadge           Currency = "timewarped_badge"
	CurrencyCoinOfAncestry            Currency = "coin_of_ancestry"
	CurrencyLoveToken                 Currency = "love_token"
	CurrencyBurningBlossom            Currency = "burning_blossom"
	CurrencyBrewfestPrizeTicket       Currency = "brewfest_prize_ticket"
	CurrencyTrickyTreat               Currency = "tricky_treat"
)

// AllCurrencies in display order.
var AllCurrencies = []Currency{
	CurrencyGold,
	CurrencyMarkOfHonor,
	CurrencyDarkmoonPrizeTicket,
	CurrencyChampionsSeal,
	CurrencyHonourboundServiceMedal,
	CurrencySeventhLegionServiceMedal,
	CurrencyTimewargedBadge,
	CurrencyCoinOfAncestry,
	CurrencyLoveToken,
	CurrencyBurningBlossom,
	CurrencyBrewfestPrizeTicket,
	CurrencyTrickyTreat,
}

// CurrencyDisplayName maps each currency to its human-readable name.
var CurrencyDisplayName = map[Currency]string{
	CurrencyGold:                      "Gold",
	CurrencyMarkOfHonor:               "Mark of Honor",
	CurrencyDarkmoonPrizeTicket:       "Darkmoon Prize Ticket",
	CurrencyChampionsSeal:             "Champion's Seal",
	CurrencyHonourboundServiceMedal:   "Honorbound Service Medal",
	CurrencySeventhLegionServiceMedal: "7th Legion Service Medal",
	CurrencyTimewargedBadge:           "Timewarped Badge",
	CurrencyCoinOfAncestry:            "Coin of Ancestry",
	CurrencyLoveToken:                 "Love Token",
	CurrencyBurningBlossom:            "Burning Blossom",
	CurrencyBrewfestPrizeTicket:       "Brewfest Prize Ticket",
	CurrencyTrickyTreat:               "Tricky Treat",
}

// CurrencyID maps each token currency to its Battle.net currency ID,
// for use by future WalletProvider implementations.
var CurrencyID = map[Currency]int{
	CurrencyMarkOfHonor:               1792,
	CurrencyDarkmoonPrizeTicket:       515,
	CurrencyChampionsSeal:             241,
	CurrencyHonourboundServiceMedal:   1716,
	CurrencySeventhLegionServiceMedal: 1717,
	CurrencyTimewargedBadge:           1166,
	CurrencyCoinOfAncestry:            47,
	CurrencyLoveToken:                 800,
	CurrencyBurningBlossom:            402,
	CurrencyBrewfestPrizeTicket:       415,
	CurrencyTrickyTreat:               356,
}

// PurchaseOption is a single currency+amount pair for buying a heirloom.
type PurchaseOption struct {
	Currency Currency `json:"currency"`
	Amount   int      `json:"amount"`
}

// HeirloomRef is a lightweight reference as returned by the heirloom index.
type HeirloomRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Key  struct {
		Href string `json:"href"`
	} `json:"key"`
}

// HeirloomIndex is the response from /data/wow/heirloom/index.
type HeirloomIndex struct {
	Heirlooms []HeirloomRef `json:"heirlooms"`
}

// HeirloomDetail is the response from /data/wow/heirloom/{id}.
type HeirloomDetail struct {
	ID   int `json:"id"`
	Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"item"`
	Source struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"source"`
	SourceDescription string `json:"source_description"`
}

// OwnedHeirloom is a single entry from the account heirloom collection.
type OwnedHeirloom struct {
	Heirloom HeirloomRef `json:"heirloom"`
	Upgrade  struct {
		Level int `json:"level"`
	} `json:"upgrade"`
}

// AccountHeirlooms is the response from /profile/user/wow/collections/heirlooms.
type AccountHeirlooms struct {
	Heirlooms []OwnedHeirloom `json:"heirlooms"`
}

// HeirloomRecord is a merged record combining cost table data with owned status.
type HeirloomRecord struct {
	ID              int
	Name            string
	Slot            string // head, shoulder, chest, legs, back, wrist, hands, waist, feet, neck, finger, trinket, main_hand, off_hand, two_hand, ranged
	ArmorType       string // cloth, leather, mail, plate, shield, staff, sword, axe, mace, dagger, bow, gun, crossbow, polearm, fist, tome
	Set             string // set name, empty if none
	IsWeapon        bool
	PurchaseOptions []PurchaseOption
	OwnedLevel      int // -1 = not owned, 0–6 = upgrade level
}

// UpgradeTier is a single upgrade step with its cost and required item.
type UpgradeTier struct {
	FromLevel int
	ToLevel   int
	Gold      int
	ItemName  string
}

// Report is the result of Calculate.
type Report struct {
	Missing    []*HeirloomRecord // not owned
	OwnedBelow []*HeirloomRecord // owned, upgrade level < 6
	OwnedMax   []*HeirloomRecord // owned, upgrade level == 6

	// MissingPurchaseTotals: for each currency, total amount to buy all missing
	// heirlooms that have a purchase option in that currency.
	MissingPurchaseTotals map[Currency]int

	// OwnedUpgradeGold: total gold to upgrade all OwnedBelow to level 6.
	OwnedUpgradeGold int

	// MissingUpgradeGold: total gold to upgrade all Missing from 0 to level 6.
	MissingUpgradeGold int

	// Wallet: known currency balances. -1 = unknown, 0 = confirmed zero, >0 = known amount.
	Wallet map[Currency]int
}

// WalletProvider abstracts currency balance lookup for token currencies.
// A stub implementation returns -1 (unknown) for all currencies.
type WalletProvider interface {
	TokenBalance(c Currency) int // -1 = unknown
}

// ProtectedCharacter holds the relevant fields from protected.json.
type ProtectedCharacter struct {
	Money int64 `json:"money"` // in copper
}
