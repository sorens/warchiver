package api

import "testing"

func TestEndpointCopyPreservesURL(t *testing.T) {
	original := &Endpoint{Url: "https://media.example/image.jpg"}

	copy := original.Copy()
	if copy.Url != original.Url {
		t.Fatalf("Copy().Url = %q, want %q", copy.Url, original.Url)
	}
}
