package api

import "testing"

func TestNewClientTrustedRegions(t *testing.T) {
	for region, want := range map[Region]string{
		RegionUS: "https://us.api.blizzard.com", RegionEU: "https://eu.api.blizzard.com",
		RegionKR: "https://kr.api.blizzard.com", RegionTW: "https://tw.api.blizzard.com",
		RegionCN: "https://gateway.battlenet.com.cn",
	} {
		c, err := NewClient(&ClientOptions{Region: region, AccessToken: "test-token"})
		if err != nil {
			t.Fatal(err)
		}
		if c.baseURL != want {
			t.Fatalf("region %q: got %q, want %q", region, c.baseURL, want)
		}
	}
	for _, region := range []Region{"", "US", "unknown", "attacker.invalid/", "localhost:443/", "us@attacker.invalid/", "us?x=", "us#"} {
		if c, err := NewClient(&ClientOptions{Region: region, AccessToken: "test-token"}); err == nil || c != nil {
			t.Errorf("accepted region %q", region)
		}
	}
	if _, err := NewClient(nil); err == nil {
		t.Error("accepted nil options")
	}
	if _, err := NewClient(&ClientOptions{Region: RegionUS}); err == nil {
		t.Error("accepted empty token")
	}
}
