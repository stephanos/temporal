package canonicaljson_test

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/world"
)

func TestStreamFieldCacheImplementationBindings(t *testing.T) {
	canonicalIdentity, worldIdentity := canonicaljson.ImplementationSHA256(), world.ImplementationSHA256()
	if canonicalIdentity == "" || worldIdentity == "" {
		t.Fatal("cache source change lost implementation binding")
	}
	t.Logf("actual source-bound canonicaljson=%s world=%s", canonicalIdentity, worldIdentity)
}
