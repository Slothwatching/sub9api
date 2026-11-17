package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Fork-local: upstream and this fork independently used snapshot v28 for
// different projections. A deployed v28 entry lacks the new Ultrafast flag.
func TestUpstreamV0222RejectsDeployedV28AuthSnapshot(t *testing.T) {
	svc := &APIKeyService{}
	entry := &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 28}}
	key, used, err := svc.applyAuthCacheEntry("synthetic-key", entry)
	require.NoError(t, err)
	require.False(t, used)
	require.Nil(t, key)
}

// Fork-local: the pricing merge must retain display-only capability facts and
// upstream explicit-price provenance on the same parsed card.
func TestUpstreamV0222PricingOverridePreservesCatalogCapabilities(t *testing.T) {
	svc := newPricingServiceWithOverride(t, `{"deepseek-v4-pro":{"input_cost_per_token":0.000003}}`)
	cards, err := svc.parsePricingData([]byte(`{"deepseek-v4-pro":{
        "input_cost_per_token":0.000001,"output_cost_per_token":0.000002,
        "max_input_tokens":128000,"max_output_tokens":8192,
        "supports_vision":true,"supports_function_calling":false}}`))
	require.NoError(t, err)
	card := cards["deepseek-v4-pro"]
	require.NotNil(t, card)
	require.InDelta(t, 0.000003, card.InputCostPerToken, 1e-12)
	require.True(t, card.priceOverrides.input)
	require.False(t, card.priceOverrides.output)
	require.NotNil(t, card.MaxInputTokens)
	require.Equal(t, 128000, *card.MaxInputTokens)
	require.NotNil(t, card.MaxOutputTokens)
	require.Equal(t, 8192, *card.MaxOutputTokens)
	require.NotNil(t, card.SupportsVision)
	require.True(t, *card.SupportsVision)
	require.NotNil(t, card.SupportsFunctionCalling)
	require.False(t, *card.SupportsFunctionCalling)
}
