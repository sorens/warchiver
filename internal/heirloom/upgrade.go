package heirloom

// Upgrade tier costs in gold per level (index 0 = tier 1→2, index 5 = tier 5→6 max).
var armorTierCosts = []int{500, 1000, 2000, 5000, 5000, 5000}
var weaponTierCosts = []int{750, 1500, 3000, 7500, 7500, 7500}

var armorTierItems = []string{
	"Ancient Armor Casing",
	"Timeworn Armor Casing",
	"Weathered Armor Casing",
	"Battle-Hardened Armor Casing",
	"Eternal Armor Casing",
	"Awakened Armor Casing",
}

var weaponTierItems = []string{
	"Ancient Scabbard",
	"Timeworn Scabbard",
	"Weathered Scabbard",
	"Battle-Hardened Scabbard",
	"Eternal Scabbard",
	"Awakened Scabbard",
}

// GoldToMax returns the total gold cost to upgrade from fromLevel to 6.
func GoldToMax(fromLevel int, weapon bool) int {
	costs := armorTierCosts
	if weapon {
		costs = weaponTierCosts
	}
	total := 0
	for i := fromLevel; i < len(costs); i++ {
		total += costs[i]
	}
	return total
}

// UpgradeTiers returns each upgrade step from fromLevel to 6.
func UpgradeTiers(fromLevel int, weapon bool) []UpgradeTier {
	costs := armorTierCosts
	items := armorTierItems
	if weapon {
		costs = weaponTierCosts
		items = weaponTierItems
	}
	var tiers []UpgradeTier
	for i := fromLevel; i < len(costs); i++ {
		tiers = append(tiers, UpgradeTier{
			FromLevel: i,
			ToLevel:   i + 1,
			Gold:      costs[i],
			ItemName:  items[i],
		})
	}
	return tiers
}
