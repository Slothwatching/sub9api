// This harness is compiled separately with the old and new repository versions.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	db, err := sql.Open("postgres", os.Getenv("COMPAT_DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := repository.ApplyMigrations(ctx, db); err != nil {
		return err
	}
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	phase := os.Args[1]
	if phase == "baseline" {
		// Synthetic rows exercise exact preservation of nonzero usage, windows,
		// soft-deleted history and an active configured limit across the purge.
		if _, err := db.ExecContext(ctx, `
INSERT INTO users (email, password_hash) VALUES ('compat@example.invalid', 'not-a-credential');
INSERT INTO user_platform_quotas (user_id, platform, daily_usage_usd, weekly_usage_usd, monthly_usage_usd, daily_window_start, weekly_window_start, monthly_window_start, deleted_at, daily_limit_usd)
SELECT id, 'openai', 1.25, 2.5, 3.75, '2026-09-14T01:00:00Z'::timestamptz, '2026-09-10T01:00:00Z'::timestamptz, '2026-09-01T01:00:00Z'::timestamptz, NULL::timestamptz, NULL::numeric FROM users WHERE email='compat@example.invalid'
UNION ALL SELECT id, 'gemini', 4, 5, 6, NULL, NULL, NULL, '2026-09-12T01:00:00Z', NULL FROM users WHERE email='compat@example.invalid'
UNION ALL SELECT id, 'anthropic', 7, 8, 9, NULL, NULL, NULL, NULL, 20 FROM users WHERE email='compat@example.invalid';
CREATE TABLE compat_expected_quota AS SELECT id, to_jsonb(q) AS row_data FROM user_platform_quotas q;
`); err != nil {
			return err
		}
		for _, name := range []string{"selected", "empty"} {
			config := domain.GroupModelsListConfig{Enabled: true}
			if name == "selected" {
				config.Models = []string{"gpt-5.5", "gpt-5.4"}
			}
			if _, err := client.Group.Create().SetName(name).SetPlatform("openai").SetModelsListConfig(config).Save(ctx); err != nil {
				return err
			}
		}
	}
	if phase != "baseline" {
		var preserved bool
		if err := db.QueryRowContext(ctx, `SELECT
  (SELECT count(*) FROM sub9api_quota_upgrade_archive a JOIN compat_expected_quota e ON a.original_id=e.id AND a.row_data=e.row_data) = 2
  AND (SELECT count(*) FROM user_platform_quotas q JOIN compat_expected_quota e ON q.id=e.id AND to_jsonb(q)=e.row_data) = 1
  AND NOT EXISTS (SELECT 1 FROM user_platform_quotas q JOIN compat_expected_quota e ON q.id=e.id WHERE q.daily_limit_usd IS NULL)
`).Scan(&preserved); err != nil {
			return err
		}
		if !preserved {
			return fmt.Errorf("quota archive or configured limit changed")
		}
	}
	if phase == "rollback" {
		if _, err := client.Group.Update().Where(group.NameEQ("selected")).SetModelsListConfig(domain.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.4"}}).Save(ctx); err != nil {
			return err
		}
		if _, err := client.Group.Create().SetName("rollback-created").SetPlatform("openai").SetModelsListConfig(domain.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.5"}}).Save(ctx); err != nil {
			return err
		}
	}
	groups, err := client.Group.Query().Where(group.NameIn("selected", "empty", "rollback-created")).Order(dbent.Asc(group.FieldID)).All(ctx)
	if err != nil {
		return err
	}
	expectedCount := 2
	if phase == "rollback" || phase == "reupgrade" {
		expectedCount = 3
	}
	if len(groups) != expectedCount {
		return fmt.Errorf("unexpected group count: %d", len(groups))
	}
	for _, g := range groups {
		config := g.ModelsListConfig
		if !config.Enabled {
			return fmt.Errorf("legacy display switch lost")
		}
		switch g.Name {
		case "empty":
			if len(config.Models) != 0 {
				return fmt.Errorf("empty display changed")
			}
		case "selected":
			expected := 2
			if expectedCount == 3 {
				expected = 1
			}
			if len(config.Models) != expected {
				return fmt.Errorf("display selection lost")
			}
		}
		if phase != "baseline" {
			var raw []byte
			if err := db.QueryRowContext(ctx, "SELECT model_allowlist FROM groups WHERE id=$1", g.ID).Scan(&raw); err != nil {
				return err
			}
			var policy struct {
				Enabled bool `json:"enabled"`
			}
			if err := json.Unmarshal(raw, &policy); err != nil {
				return err
			}
			if policy.Enabled {
				return fmt.Errorf("legacy display became request admission")
			}
		}
	}
	fmt.Printf("COMPAT_OK phase=%s groups=%d\n", phase, len(groups))
	return nil
}
