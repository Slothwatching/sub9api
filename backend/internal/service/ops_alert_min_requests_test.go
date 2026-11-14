//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// D144: exercise the actual dashboard-to-alert path for every S+E ratio.
func TestComputeRuleMetricMinRequests(t *testing.T) {
	for _, metric := range []string{"error_rate", "success_rate", "upstream_error_rate"} {
		t.Run(metric, func(t *testing.T) {
			for _, threshold := range []int64{10, 3, 1} {
				for _, n := range []int64{0, threshold - 1, threshold, threshold + 1} {
					overview := &OpsDashboardOverview{RequestCountSLA: n, SLA: 0.75, ErrorRate: 0.25, UpstreamErrorRate: 0.125}
					svc := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{overview: overview}}
					rule := &OpsAlertRule{MetricType: metric}
					if threshold != 10 {
						rule.Filters = map[string]any{"min_requests": float64(threshold)}
					}
					got, ok := svc.computeRuleMetric(context.Background(), rule, nil, time.Now().Add(-time.Minute), time.Now(), "", nil)
					require.Equal(t, n >= threshold, ok, "threshold=%d requests=%d", threshold, n)
					if ok {
						require.Equal(t, map[string]float64{"error_rate": 25, "success_rate": 75, "upstream_error_rate": 12.5}[metric], got)
					} else {
						require.Zero(t, got)
					}
				}
			}
		})
	}
}

func TestOpsAlertMinRequests(t *testing.T) {
	require.EqualValues(t, 10, opsAlertMinRequests(nil))
	for _, value := range []any{nil, 0, -1, 0.5, 1.5, "bad", "0", "-1", "1.5", true, math.NaN(), math.Inf(1), float64(math.MaxInt32) + 1, int64(math.MaxInt64)} {
		require.EqualValues(t, 10, opsAlertMinRequests(map[string]any{"min_requests": value}), "value=%v", value)
	}
	for _, value := range []any{3, int64(3), float64(3), " 3 "} {
		require.EqualValues(t, 3, opsAlertMinRequests(map[string]any{"min_requests": value}))
	}
}

// Event methods intentionally remain unimplemented: touching them would panic,
// proving low samples cannot create or resolve an event (even if one is active).
type lowSampleAlertRepo struct {
	stubOpsRepo
	rule *OpsAlertRule
}

func (r *lowSampleAlertRepo) ListAlertRules(context.Context) ([]*OpsAlertRule, error) {
	return []*OpsAlertRule{r.rule}, nil
}
func (r *lowSampleAlertRepo) GetLatestSystemMetrics(context.Context, int) (*OpsSystemMetricsSnapshot, error) {
	return nil, nil
}
func (r *lowSampleAlertRepo) UpsertJobHeartbeat(context.Context, *OpsUpsertJobHeartbeatInput) error {
	return nil
}

func TestLowSampleAlertSkipsEventLifecycle(t *testing.T) {
	for _, n := range []int64{0, 1, 9} {
		for _, rate := range []float64{0, 1} {
			repo := &lowSampleAlertRepo{stubOpsRepo: stubOpsRepo{overview: &OpsDashboardOverview{RequestCountSLA: n, ErrorRate: rate}}, rule: &OpsAlertRule{ID: 144, Enabled: true, MetricType: "error_rate", Operator: ">", Threshold: 20, WindowMinutes: 1, SustainedMinutes: 1}}
			svc := &OpsAlertEvaluatorService{opsRepo: repo, ruleStates: map[int64]*opsAlertRuleState{144: {ConsecutiveBreaches: 5}}}
			svc.evaluateOnce(time.Minute)
			require.Zero(t, svc.ruleStates[144].ConsecutiveBreaches)
			require.False(t, svc.ruleStates[144].LastEvaluatedAt.IsZero())
		}
	}
}
