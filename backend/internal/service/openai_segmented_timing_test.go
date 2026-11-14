package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
	"github.com/stretchr/testify/require"
)

func TestOpenAIRecordUsageSegmentedTimingPreservesBilling(t *testing.T) {
	var costs [2]float64
	var commands [2]*UsageBillingCommand
	for i := 0; i < 2; i++ {
		usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
		billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
		svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
		legacy := 12
		result := &OpenAIForwardResult{RequestID: "segmented", Model: "gpt-5.1", Duration: time.Second, FirstTokenMs: &legacy, Usage: OpenAIUsage{InputTokens: 20, OutputTokens: 10}}
		if i == 1 {
			_, clock := requestlatency.New(context.Background())
			clock.BeginAttempt()(true)
			clock.BeginAttempt()(false)
			clock.Mark(requestlatency.ClientFirstContent)
			clock.Mark(requestlatency.UserDuration)
			result.Latency = clock.Snapshot()
		}
		err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{Result: result, APIKey: &APIKey{ID: 10, Group: &Group{RateMultiplier: 1}}, User: &User{ID: 20}, Account: &Account{ID: 30, Type: AccountTypeAPIKey}})
		require.NoError(t, err)
		require.NotNil(t, usageRepo.lastLog)
		require.Equal(t, result.Latency, usageRepo.lastLog.Snapshot)
		require.Equal(t, 12, *usageRepo.lastLog.FirstTokenMs)
		require.Equal(t, 1000, *usageRepo.lastLog.DurationMs)
		costs[i] = usageRepo.lastLog.ActualCost
		commands[i] = billingRepo.lastCmd
	}
	require.Greater(t, costs[0], 0.0)
	require.Equal(t, costs[0], costs[1])
	require.Equal(t, commands[0], commands[1])
}

func TestOpenAIUpstreamSegmentedTimingObservation(t *testing.T) {
	for _, payload := range []string{
		"data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n",
		"data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n",
	} {
		ctx, clock := requestlatency.New(context.Background())
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://mock.invalid", nil)
		require.NoError(t, err)
		upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(payload))}}
		svc := &OpenAIGatewayService{httpUpstream: upstream}
		resp, err := svc.doOpenAIUpstream(req, "", &Account{ID: 1})
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, payload, string(body))
		require.NoError(t, resp.Body.Close())
		got := clock.Snapshot()
		require.NotNil(t, got.UpstreamFirstByteMs)
		require.NotNil(t, got.FirstDataMs)
		require.NotNil(t, got.FirstSemanticMs)
		require.NotNil(t, got.FirstVisibleContentMs)
		require.Nil(t, got.ClientFirstContentMs)
		require.Nil(t, got.UpstreamConnectMs, "stub transport emits no trace callbacks")
	}
}

type segmentedTimingPluginManagerStub struct {
	response *http.Response
	calls    int
}

func (p *segmentedTimingPluginManagerStub) RoundTripOpenAIOAuth(context.Context, *http.Request, string, *Account) (*http.Response, bool, error) {
	p.calls++
	return p.response, true, nil
}

func (*segmentedTimingPluginManagerStub) ShouldRouteOpenAIOAuth(*Account) bool {
	return true
}

func TestOpenAIUpstreamSegmentedTimingPluginObservation(t *testing.T) {
	for name, payload := range map[string]string{
		"responses": "data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n",
		"chat":      "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			ctx, clock := requestlatency.New(context.Background())
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://mock.invalid", nil)
			require.NoError(t, err)
			plugin := &segmentedTimingPluginManagerStub{response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(payload)),
			}}
			upstream := &httpUpstreamRecorder{}
			svc := &OpenAIGatewayService{pluginManager: plugin, httpUpstream: upstream}
			resp, err := svc.doOpenAIUpstream(req, "", &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth})
			require.NoError(t, err)
			require.NotNil(t, resp)
			defer resp.Body.Close()
			require.Same(t, plugin.response, resp)
			require.Equal(t, 1, plugin.calls)
			require.Empty(t, upstream.requests, "handled plugin request must bypass HTTP upstream")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, payload, string(body))
			got := clock.Snapshot()
			require.NotNil(t, got.UpstreamFirstByteMs)
			require.NotNil(t, got.FirstDataMs)
			require.NotNil(t, got.FirstSemanticMs)
			require.NotNil(t, got.FirstVisibleContentMs)
		})
	}
}
