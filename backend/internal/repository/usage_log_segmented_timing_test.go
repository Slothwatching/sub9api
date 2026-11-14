//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogSegmentedTimingSQLRoundTrip(t *testing.T) {
	for _, present := range []bool{false, true} {
		db, mock := newSQLMock(t)
		repo := &usageLogRepository{sql: db}
		log := &service.UsageLog{UserID: 1, APIKeyID: 2, AccountID: 3, RequestID: "timing-mock", Model: "gpt-5.1", CreatedAt: time.Now()}
		if present {
			_, clock := requestlatency.New(context.Background())
			clock.Mark(requestlatency.FirstData)
			clock.Mark(requestlatency.FirstSemantic)
			clock.Mark(requestlatency.FirstVisibleContent)
			clock.Mark(requestlatency.ClientFirstContent)
			clock.Mark(requestlatency.UserDuration)
			clock.Add(requestlatency.UserQueue, 3*time.Millisecond)
			clock.Add(requestlatency.AccountQueue, 4*time.Millisecond)
			clock.Add(requestlatency.AccountSelection, 5*time.Millisecond)
			clock.Add(requestlatency.UpstreamConnect, 6*time.Millisecond)
			clock.Add(requestlatency.UpstreamTLS, 7*time.Millisecond)
			clock.Mark(requestlatency.UpstreamFirstByte)
			clock.BeginAttempt()(true)
			clock.BeginAttempt()(false)
			log.Snapshot = clock.Snapshot()
		}
		prepared := prepareUsageLogInsert(log)
		values := []driver.Value{int64(99)}
		for _, arg := range prepared.args {
			v, err := driver.DefaultParameterConverter.ConvertValue(arg)
			require.NoError(t, err)
			values = append(values, v)
		}
		mock.ExpectQuery("SELECT .* FROM usage_logs WHERE id").WithArgs(99).WillReturnRows(sqlmock.NewRows(strings.Split(usageLogSelectColumns, ", ")).AddRow(values...))
		got, err := repo.GetByID(context.Background(), 99)
		require.NoError(t, err)
		require.Equal(t, log.Snapshot, got.Snapshot)
		require.Nil(t, got.FirstTokenMs, "legacy field is independent")
		require.NoError(t, mock.ExpectationsWereMet())
	}
}
