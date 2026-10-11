package data

// ValueDisplay is a numeric value paired with a human-readable string.
type ValueDisplay struct {
	Value         int    `json:"value"`
	DisplayString string `json:"display_string"`
}

type ItemMediaRef struct {
	Key BasicLink `json:"key"`
	ID  int       `json:"id"`
}

type StatColor struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}

type StatDisplay struct {
	DisplayString string    `json:"display_string"`
	Color         StatColor `json:"color"`
}

type ItemStat struct {
	Type      TypeRef     `json:"type"`
	Value     int         `json:"value"`
	IsNegated bool        `json:"is_negated"`
	Display   StatDisplay `json:"display"`
}

type ItemTransmog struct {
	Item                     ItemIDRef `json:"item"`
	DisplayString            string    `json:"display_string"`
	ItemModifiedAppearanceID int       `json:"item_modified_appearance_id"`
}

type RequirementGroup struct {
	Links         []RefData `json:"links"`
	DisplayString string    `json:"display_string"`
}

type FactionRequirement struct {
	Value         TypeRef `json:"value"`
	DisplayString string  `json:"display_string"`
}

type ItemRequirements struct {
	Level           *ValueDisplay       `json:"level,omitempty"`
	PlayableClasses *RequirementGroup   `json:"playable_classes,omitempty"`
	PlayableRaces   *RequirementGroup   `json:"playable_races,omitempty"`
	Faction         *FactionRequirement `json:"faction,omitempty"`
}

type SellPricePart struct {
	DisplayString string    `json:"display_string"`
	Color         StatColor `json:"color"`
}

type SellPriceDisplay struct {
	Header SellPricePart `json:"header"`
	Gold   SellPricePart `json:"gold"`
	Silver SellPricePart `json:"silver"`
	Copper SellPricePart `json:"copper"`
}

type ItemSellPrice struct {
	Value        int              `json:"value"`
	DisplayParts SellPriceDisplay `json:"display_parts"`
}

type ItemSetEffect struct {
	DisplayString string `json:"display_string"`
	RequiredCount int    `json:"required_count"`
	IsActive      bool   `json:"is_active"`
}

type ItemSetItem struct {
	Item ItemIDRef `json:"item"`
	Slot TypeRef   `json:"slot"`
}

type EquippedItemSet struct {
	ItemSet       RefData         `json:"item_set"`
	Items         []ItemSetItem   `json:"items"`
	Effects       []ItemSetEffect `json:"effects"`
	DisplayString string          `json:"display_string"`
}

// EquipmentResponse is the top-level shape of equipment/equipment.json.
type EquipmentResponse struct {
	EquippedItems    []EquippedItem    `json:"equipped_items"`
	EquippedItemSets []EquippedItemSet `json:"equipped_item_sets"`
}

// EquippedItem is a single item in the equipped_items array.
type EquippedItem struct {
	Slot                 TypeRef          `json:"slot"`
	Item                 ItemIDRef        `json:"item"`
	Name                 string           `json:"name"`
	Quality              TypeRef          `json:"quality"`
	Level                ValueDisplay     `json:"level"`
	Quantity             int              `json:"quantity"`
	Context              int              `json:"context"`
	BonusList            []int            `json:"bonus_list"`
	InventoryType        TypeRef          `json:"inventory_type"`
	Binding              TypeRef          `json:"binding"`
	Armor                ValueDisplay     `json:"armor"`
	Stats                []ItemStat       `json:"stats"`
	SellPrice            ItemSellPrice    `json:"sell_price"`
	Requirements         ItemRequirements `json:"requirements"`
	Durability           ValueDisplay     `json:"durability"`
	ItemClass            RefData          `json:"item_class"`
	ItemSubclass         RefData          `json:"item_subclass"`
	Enchantments         []Enchantment    `json:"enchantments"`
	Sockets              []Socket         `json:"sockets"`
	Transmog             *ItemTransmog    `json:"transmog,omitempty"`
	Media                ItemMediaRef     `json:"media"`
	ModifiedAppearanceID int              `json:"modified_appearance_id"`
}

// TypeRef is a {type, name} pair used for slots, socket types, enchantment slots, etc.
type TypeRef struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// ItemIDRef is a minimal item reference carrying only the item ID.
type ItemIDRef struct {
	ID int `json:"id"`
}

// Enchantment is a single enchantment applied to an item.
type Enchantment struct {
	EnchantmentID   int      `json:"enchantment_id"`
	EnchantmentSlot TypeRef  `json:"enchantment_slot"`
	DisplayString   string   `json:"display_string"`
	SourceItem      *RefData `json:"source_item,omitempty"`
}

// Socket is a gem socket on an item.
type Socket struct {
	Item          ItemIDRef `json:"item"`
	SocketType    TypeRef   `json:"socket_type"`
	DisplayString string    `json:"display_string"`
}
