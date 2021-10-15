package metadata

import (
	"context"
	"testing"
)

func TestGoogleGroups(t *testing.T) {
	ctx := context.Background()
	gg, err := NewGoogleGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}

	gg.ForUser(ctx, "thomas.siegfried.krampl@nav.no")
}

func TestGoogleProjects(t *testing.T) {
	ctx := context.Background()
	gg, err := NewGoogleGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}

	gg.Projects(ctx, "ya29.a0ARrdaM9roSutVQjog0vLF4TwK0iwMXXloFV4qRBFUr8qd0bpdWqv9Q5xHcU8OfrHBG81ZhOfscpkGhpnqtrOmfoMCyQi1dZjIh5Niw5GHC1qTJhcg6gjqBGD5SMJ5-4U34gMTIWNZcrGEvUAdPPfqYMRvFqJ")
}
