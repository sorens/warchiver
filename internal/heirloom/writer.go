package heirloom

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Write creates all heirloom CSV files in outDir.
func Write(outDir string, r *Report) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	writers := []func(string, *Report) error{
		writeSummary,
		writeMissing,
		writeUpgradeExisting,
		writeUpgradeMissing,
		writeByWallet,
	}
	for _, w := range writers {
		if err := w(outDir, r); err != nil {
			return err
		}
	}
	for _, c := range AllCurrencies {
		if err := writeByCurrency(outDir, r, c); err != nil {
			return err
		}
	}
	return nil
}

// withCSV handles the open/header/flush/close lifecycle for a CSV file under
// outDir. fn writes the body rows; the header (when non-nil) is written
// before fn runs. Flush+Close happen via defer in LIFO order so the file is
// closed only after the writer has flushed.
func withCSV(outDir, name string, header []string, fn func(*csv.Writer) error) error {
	f, err := os.Create(filepath.Join(outDir, name))
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	defer f.Close()
	if header != nil {
		w.Write(header)
	}
	if err := fn(w); err != nil {
		return err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return f.Close()
}

func fmtGold(g int) string {
	return fmt.Sprintf("%d", g)
}

func fmtAmount(v int) string {
	if v < 0 {
		return "-"
	}
	return strconv.Itoa(v)
}

func fmtCost(amount int) string {
	if amount == 0 {
		return "?"
	}
	return strconv.Itoa(amount)
}

func itemType(rec *HeirloomRecord) string {
	parts := []string{rec.Slot}
	if rec.ArmorType != "" {
		parts = append(parts, rec.ArmorType)
	}
	return strings.Join(parts, "/")
}

func purchaseCostFor(rec *HeirloomRecord, c Currency) string {
	for _, opt := range rec.PurchaseOptions {
		if opt.Currency == c {
			return fmtCost(opt.Amount)
		}
	}
	return ""
}

func writeSummary(outDir string, r *Report) error {
	header := []string{"Category", "Count", "Gold"}
	for _, c := range AllCurrencies {
		if c != CurrencyGold {
			header = append(header, CurrencyDisplayName[c])
		}
	}
	return withCSV(outDir, "summary.csv", header, func(w *csv.Writer) error {
		row := func(label string, count int, goldAmt int, extra map[Currency]int) {
			r := []string{label, strconv.Itoa(count), fmtGold(goldAmt)}
			for _, c := range AllCurrencies {
				if c != CurrencyGold {
					if v, ok := extra[c]; ok {
						r = append(r, fmtCost(v))
					} else {
						r = append(r, "")
					}
				}
			}
			w.Write(r)
		}
		row("Missing (purchase only)", len(r.Missing), r.MissingPurchaseTotals[CurrencyGold], r.MissingPurchaseTotals)
		row("Missing (upgrade to max)", len(r.Missing), r.MissingUpgradeGold, nil)
		row("Owned (upgrade to max)", len(r.OwnedBelow), r.OwnedUpgradeGold, nil)
		row("Already maxed", len(r.OwnedMax), 0, nil)
		row("Total heirlooms", len(r.Missing)+len(r.OwnedBelow)+len(r.OwnedMax), 0, nil)
		return nil
	})
}

func writeMissing(outDir string, r *Report) error {
	header := []string{"ID", "Name", "Set", "Slot", "Type", "Purchase Currency", "Purchase Cost", "Upgrade Cost (gold)", "Total to Max (gold)"}
	return withCSV(outDir, "missing.csv", header, func(w *csv.Writer) error {
		for _, rec := range r.Missing {
			for _, opt := range rec.PurchaseOptions {
				upgGold := GoldToMax(0, rec.IsWeapon)
				totalGold := upgGold
				if opt.Currency == CurrencyGold && opt.Amount > 0 {
					totalGold += opt.Amount
				}
				w.Write([]string{
					strconv.Itoa(rec.ID),
					rec.Name,
					rec.Set,
					rec.Slot,
					itemType(rec),
					CurrencyDisplayName[opt.Currency],
					fmtCost(opt.Amount),
					fmtGold(upgGold),
					fmtGold(totalGold),
				})
			}
			if len(rec.PurchaseOptions) == 0 {
				upgGold := GoldToMax(0, rec.IsWeapon)
				w.Write([]string{
					strconv.Itoa(rec.ID),
					rec.Name,
					rec.Set,
					rec.Slot,
					itemType(rec),
					"unknown",
					"?",
					fmtGold(upgGold),
					"?",
				})
			}
		}
		return nil
	})
}

func writeUpgradeExisting(outDir string, r *Report) error {
	header := []string{"ID", "Name", "Set", "Slot", "Type", "Current Level", "Gold to Max", "Upgrade Items"}
	return withCSV(outDir, "upgrade-existing.csv", header, func(w *csv.Writer) error {
		writeRows := func(recs []*HeirloomRecord) {
			for _, rec := range recs {
				tiers := UpgradeTiers(rec.OwnedLevel, rec.IsWeapon)
				items := make([]string, len(tiers))
				for i, t := range tiers {
					items[i] = t.ItemName
				}
				w.Write([]string{
					strconv.Itoa(rec.ID),
					rec.Name,
					rec.Set,
					rec.Slot,
					itemType(rec),
					strconv.Itoa(rec.OwnedLevel),
					fmtGold(GoldToMax(rec.OwnedLevel, rec.IsWeapon)),
					strings.Join(items, "; "),
				})
			}
		}
		writeRows(r.OwnedBelow)
		writeRows(r.OwnedMax)
		return nil
	})
}

func writeUpgradeMissing(outDir string, r *Report) error {
	header := []string{"ID", "Name", "Set", "Slot", "Type", "Purchase Currency", "Purchase Cost", "Upgrade Cost (gold)", "Grand Total (gold)"}
	return withCSV(outDir, "upgrade-missing.csv", header, func(w *csv.Writer) error {
		for _, rec := range r.Missing {
			upgGold := GoldToMax(0, rec.IsWeapon)
			if len(rec.PurchaseOptions) == 0 {
				w.Write([]string{
					strconv.Itoa(rec.ID), rec.Name, rec.Set, rec.Slot, itemType(rec),
					"unknown", "?", fmtGold(upgGold), "?",
				})
				continue
			}
			for _, opt := range rec.PurchaseOptions {
				totalGold := upgGold
				if opt.Currency == CurrencyGold && opt.Amount > 0 {
					totalGold += opt.Amount
				}
				w.Write([]string{
					strconv.Itoa(rec.ID), rec.Name, rec.Set, rec.Slot, itemType(rec),
					CurrencyDisplayName[opt.Currency],
					fmtCost(opt.Amount),
					fmtGold(upgGold),
					fmtGold(totalGold),
				})
			}
		}
		return nil
	})
}

func writeByCurrency(outDir string, r *Report, c Currency) error {
	filename := "by-" + string(c) + ".csv"
	filename = strings.ReplaceAll(filename, "_", "-")

	var rows []*HeirloomRecord
	for _, rec := range r.Missing {
		if purchaseCostFor(rec, c) != "" {
			rows = append(rows, rec)
		}
	}
	for _, rec := range r.OwnedBelow {
		if purchaseCostFor(rec, c) != "" {
			rows = append(rows, rec)
		}
	}
	for _, rec := range r.OwnedMax {
		if purchaseCostFor(rec, c) != "" {
			rows = append(rows, rec)
		}
	}

	if len(rows) == 0 {
		return nil
	}

	header := []string{"ID", "Name", "Set", "Slot", "Type", "Status", "Purchase Cost", "Upgrade Level", "Gold to Max", "Total Gold"}
	return withCSV(outDir, filename, header, func(w *csv.Writer) error {
		for _, rec := range rows {
			status := "missing"
			if rec.OwnedLevel >= 0 {
				status = fmt.Sprintf("owned (level %d)", rec.OwnedLevel)
			}
			if rec.OwnedLevel == 6 {
				status = "maxed"
			}

			cost := purchaseCostFor(rec, c)
			upgGold := GoldToMax(max(rec.OwnedLevel, 0), rec.IsWeapon)
			totalGold := upgGold
			if c == CurrencyGold {
				if rec.OwnedLevel < 0 {
					amt := 0
					for _, opt := range rec.PurchaseOptions {
						if opt.Currency == CurrencyGold {
							amt = opt.Amount
							break
						}
					}
					if amt > 0 {
						totalGold += amt
					}
				}
			}

			w.Write([]string{
				strconv.Itoa(rec.ID),
				rec.Name,
				rec.Set,
				rec.Slot,
				itemType(rec),
				status,
				cost,
				strconv.Itoa(max(rec.OwnedLevel, 0)),
				fmtGold(upgGold),
				fmtGold(totalGold),
			})
		}
		return nil
	})
}

func writeByWallet(outDir string, r *Report) error {
	header := []string{"Currency", "Have", "Need to Buy Missing", "Need to Max Owned", "Need to Max All", "Deficit"}
	return withCSV(outDir, "by-wallet.csv", header, func(w *csv.Writer) error {
		for _, c := range AllCurrencies {
			have := r.Wallet[c]
			needBuy := r.MissingPurchaseTotals[c]
			needUpgradeOwned := 0
			needUpgradeMissing := 0
			if c == CurrencyGold {
				needUpgradeOwned = r.OwnedUpgradeGold
				needUpgradeMissing = r.MissingUpgradeGold
			}
			needMax := needBuy + needUpgradeOwned + needUpgradeMissing

			deficit := ""
			if have >= 0 && needMax > 0 {
				d := needMax - have
				if d > 0 {
					deficit = strconv.Itoa(d)
				} else {
					deficit = "0 (sufficient)"
				}
			} else if have < 0 {
				deficit = "-"
			}

			w.Write([]string{
				CurrencyDisplayName[c],
				fmtAmount(have),
				fmtCost(needBuy),
				fmtGold(needUpgradeOwned),
				fmtCost(needMax),
				deficit,
			})
		}
		return nil
	})
}
