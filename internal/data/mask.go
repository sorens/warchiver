package data

type DataMask uint8

const (
	DataMaskNone DataMask = 0

	DataMaskCharacterEntry = 1 << 0
	DataMaskSimc           = 1 << 1
	DataMaskAll            = 1 << 2
	DataMaskDefault        = DataMaskCharacterEntry
)
