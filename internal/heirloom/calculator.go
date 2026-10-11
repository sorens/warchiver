package heirloom

import (
	"sort"
)

type stubWallet struct{}

func (stubWallet) TokenBalance(Currency) int { return -1 }

// Calculate builds a Report from the heirloom index, optional detail map,
// owned upgrade levels, total gold in copper across all characters, and
// an optional WalletProvider for token currencies (nil uses the stub).
func Calculate(
	index *HeirloomIndex,
	details map[int]*HeirloomDetail,
	owned map[int]int,
	goldCopper int64,
	wallet WalletProvider,
) *Report {
	if wallet == nil {
		wallet = stubWallet{}
	}

	r := &Report{
		MissingPurchaseTotals: make(map[Currency]int),
		Wallet:                make(map[Currency]int),
	}

	if goldCopper < 0 {
		r.Wallet[CurrencyGold] = -1
	} else {
		r.Wallet[CurrencyGold] = int(goldCopper / 10000)
	}
	for _, c := range AllCurrencies {
		if c == CurrencyGold {
			continue
		}
		r.Wallet[c] = wallet.TokenBalance(c)
	}

	for i := range index.Heirlooms {
		ref := &index.Heirlooms[i]
		entry := CostEntry(ref.ID)

		rec := &HeirloomRecord{
			ID:         ref.ID,
			Name:       ref.Name,
			OwnedLevel: -1,
		}
		if entry != nil {
			rec.Slot = entry.Slot
			rec.ArmorType = entry.ArmorType
			rec.Set = entry.Set
			rec.IsWeapon = entry.IsWeapon
			rec.PurchaseOptions = entry.PurchaseOptions
		}
		// Detail from individual API endpoint (source description, etc.)
		// Reserved for future enrichment; slot/type data from costs.json is authoritative.
		_ = details

		if level, owns := owned[ref.ID]; owns {
			rec.OwnedLevel = level
		}

		switch {
		case rec.OwnedLevel < 0:
			r.Missing = append(r.Missing, rec)
			for _, opt := range rec.PurchaseOptions {
				if opt.Amount > 0 {
					r.MissingPurchaseTotals[opt.Currency] += opt.Amount
				}
			}
			r.MissingUpgradeGold += GoldToMax(0, rec.IsWeapon)
		case rec.OwnedLevel < 6:
			r.OwnedBelow = append(r.OwnedBelow, rec)
			r.OwnedUpgradeGold += GoldToMax(rec.OwnedLevel, rec.IsWeapon)
		default:
			r.OwnedMax = append(r.OwnedMax, rec)
		}
	}

	byName := func(recs []*HeirloomRecord) {
		sort.Slice(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	}
	byName(r.Missing)
	byName(r.OwnedBelow)
	byName(r.OwnedMax)

	return r
}
