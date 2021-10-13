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
