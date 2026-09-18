//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// Block the second query until a historical deletion commits. A single read
// must not combine the old watermark with data from after the deletion.
func TestGroupUsageSummaryConcurrentHistoricalDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	schema := createGroupUsageRollupTriggerTestSchema(t, ctx, false)
	seed := beginGroupUsageRollupTriggerTestTx(t, ctx, schema)
	defer func() { _ = seed.Rollback() }()
	_, err := seed.ExecContext(ctx, `
        INSERT INTO groups (id) VALUES (10), (20);
        INSERT INTO users (id) VALUES (1), (2);
        INSERT INTO usage_logs (id, user_id, group_id, actual_cost, created_at)
        VALUES (1, 1, 10, 1.25, TIMESTAMPTZ '2020-01-02 12:00:00Z');
    `)
	require.NoError(t, err)
	today := service.GroupUsageTodayStart(time.Now())
	_, err = seed.ExecContext(ctx, "INSERT INTO usage_logs (id, user_id, group_id, actual_cost, created_at) VALUES (2, 2, 20, 2.5, $1)", today.Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, newDashboardAggregationRepositoryWithSQL(seed).SyncGroupUsageRollups(ctx, today))
	require.NoError(t, seed.Commit())

	conn, err := integrationDB.Conn(ctx)
	require.NoError(t, err)
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "RESET search_path")
		_ = conn.Close()
	}()
	_, err = conn.ExecContext(ctx, "SET search_path TO "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	var pid int
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid))
	repo := newUsageLogRepositoryWithSQL(nil, conn)

	blocker := beginGroupUsageRollupTriggerTestTx(t, ctx, schema)
	defer func() { _ = blocker.Rollback() }()
	_, err = blocker.ExecContext(ctx, "LOCK TABLE usage_group_daily_rollups IN ACCESS EXCLUSIVE MODE")
	require.NoError(t, err)
	type outcome struct {
		rows []usagestats.GroupUsageSummary
		err  error
	}
	finished := make(chan outcome, 1)
	go func() {
		rows, readErr := repo.GetAllGroupUsageSummary(ctx, today)
		finished <- outcome{rows, readErr}
	}()
	require.Eventually(t, func() bool {
		var blocked bool
		queryErr := integrationDB.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock')", pid).Scan(&blocked)
		return queryErr == nil && blocked
	}, 5*time.Second, 10*time.Millisecond)

	// Delete historical and current usage together. A torn read combines the
	// stale historical bucket (1.25) with the now-empty current tail (0).
	deleteTx := beginGroupUsageRollupTriggerTestTx(t, ctx, schema)
	defer func() { _ = deleteTx.Rollback() }()
	_, err = deleteTx.ExecContext(ctx, "DELETE FROM users")
	require.NoError(t, err)
	require.NoError(t, deleteTx.Commit())
	require.NoError(t, blocker.Commit())
	first := <-finished
	require.NoError(t, first.err)
	require.Len(t, first.rows, 2)
	require.InDelta(t, 1.25, first.rows[0].TotalCost, 1e-9, "in-flight summary must retain its pre-delete snapshot")
	require.InDelta(t, 2.5, first.rows[1].TotalCost, 1e-9, "both groups must use the same pre-delete snapshot")
	rows, err := repo.GetAllGroupUsageSummary(ctx, today)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Zero(t, rows[0].TotalCost, "the next summary must observe the committed delete")
	require.Zero(t, rows[1].TotalCost)
}
