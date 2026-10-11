//go:build !warx

package main

import "github.com/sorens/warchiver/pkg/features"

func archiveFeatureProvider() features.Provider {
	return features.Disabled{}
}
