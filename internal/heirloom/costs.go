package heirloom

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/costs.json
var costsData []byte

// CostTable is the structure of data/costs.json.
type CostTable struct {
	Version   string              `json:"version"`
	Updated   string              `json:"updated"`
	Heirlooms []HeirloomCostEntry `json:"heirlooms"`
}

// HeirloomCostEntry is one row in the embedded cost table.
type HeirloomCostEntry struct {
	ID              int              `json:"id"`
	Name            string           `json:"name"`
	Slot            string           `json:"slot"`
	ArmorType       string           `json:"armor_type"`
	Set             string           `json:"set"`
	IsWeapon        bool             `json:"is_weapon"`
	PurchaseOptions []PurchaseOption `json:"purchase_options"`
}

var costTable CostTable
var costIndex map[int]*HeirloomCostEntry

func init() {
	if err := json.Unmarshal(costsData, &costTable); err != nil {
		panic(fmt.Sprintf("heirloom: failed to parse costs.json: %v", err))
	}
	costIndex = make(map[int]*HeirloomCostEntry, len(costTable.Heirlooms))
	for i := range costTable.Heirlooms {
		e := &costTable.Heirlooms[i]
		costIndex[e.ID] = e
	}
}

// CostEntry returns the cost entry for a heirloom ID, or nil if not in the table.
func CostEntry(id int) *HeirloomCostEntry {
	return costIndex[id]
}

// TableVersion returns the WoW version string from costs.json.
func TableVersion() string {
	return costTable.Version
}
