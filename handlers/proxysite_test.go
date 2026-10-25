package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxySiteUnreachableRemoteRulesetDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		ProxySite("http://127.0.0.1:1/ruleset.yaml")
	})
}
