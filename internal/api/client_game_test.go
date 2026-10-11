package api

import "testing"

func TestEndpointNamespaceUsesGameNamespace(t *testing.T) {
	retail := &Client{region: RegionUS, game: GameRetail}
	classic := &Client{region: RegionUS, game: GameClassic}
	ep := &Endpoint{Namespace: "profile-%s"}
	if got := retail.endpointNamespace(ep); got != "profile-us" {
		t.Fatalf("retail namespace = %q", got)
	}
	if got := classic.endpointNamespace(ep); got != "profile-classic-us" {
		t.Fatalf("classic namespace = %q", got)
	}
}
