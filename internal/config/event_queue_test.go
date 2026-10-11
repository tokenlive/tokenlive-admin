package config

import (
	"testing"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
)

func TestEventQueueTopicDefaultMatchesGatewayContract(t *testing.T) {
	cfg := new(Config)
	require.NoError(t, defaults.Set(cfg))
	require.Equal(t, gatewaycontract.Keys.Events.Policy, cfg.Storage.EventQueue.Topic)
}
