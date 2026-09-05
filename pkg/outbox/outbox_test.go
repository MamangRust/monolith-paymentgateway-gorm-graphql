package outbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// sqlRecorder is a gorm logger that captures every explained SQL statement.
// GORM logs the final SQL through Trace even in DryRun mode, which is how the
// tests below assert on query shape without a live database.
type sqlRecorder struct {
	sqls []string
}

func (r *sqlRecorder) LogMode(gormlogger.LogLevel) gormlogger.Interface { return r }
func (r *sqlRecorder) Info(context.Context, string, ...interface{})     {}
func (r *sqlRecorder) Warn(context.Context, string, ...interface{})     {}
func (r *sqlRecorder) Error(context.Context, string, ...interface{})    {}
func (r *sqlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	if sql, _ := fc(); sql != "" {
		r.sqls = append(r.sqls, sql)
	}
}

func (r *sqlRecorder) last() string {
	if len(r.sqls) == 0 {
		return ""
	}
	return r.sqls[len(r.sqls)-1]
}

// dryRunDB returns a GORM session that builds SQL without executing it, so
// tests can assert on the exact query shape. DisableAutomaticPing prevents
// gorm.Open from dialing the placeholder DSN; DryRun at the config level
// survives WithContext clones.
func dryRunDB(t *testing.T) (*gorm.DB, *sqlRecorder) {
	t.Helper()
	rec := &sqlRecorder{}
	db, err := gorm.Open(gormpostgres.Open("postgres://localhost:5432/unused"), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		DryRun:                 true,
		Logger:                 rec,
	})
	require.NoError(t, err)
	return db, rec
}

func mustDryRun(t *testing.T) *gorm.DB {
	t.Helper()
	db, _ := dryRunDB(t)
	return db
}

func TestEnqueueRejectsIncompleteEvent(t *testing.T) {
	for name, event := range map[string]Event{
		"missing key":     {Topic: "topic", Payload: []byte("x")},
		"missing topic":   {EventKey: "key", Payload: []byte("x")},
		"missing payload": {EventKey: "key", Topic: "topic"},
		"invalid json":    {EventKey: "key", Topic: "topic", Payload: []byte("not-json")},
	} {
		t.Run(name, func(t *testing.T) {
			if err := Enqueue(context.Background(), mustDryRun(t), event); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("expected ErrInvalidEvent, got %v", err)
			}
		})
	}
}

func TestEnqueuePersistsEventIdempotently(t *testing.T) {
	db, rec := dryRunDB(t)
	event := Event{EventKey: "topup:42", Topic: "topup.created", Key: "42", Payload: []byte(`{"id":42}`)}
	require.NoError(t, Enqueue(context.Background(), db, event))

	sql := rec.last()
	assert.Contains(t, sql, "INSERT INTO outbox_events")
	assert.Contains(t, sql, "ON CONFLICT (event_key) DO NOTHING")
}

func TestNewRelayRejectsNilDependencies(t *testing.T) {
	if _, err := NewRelay(nil, &fakePublisher{}, RelayConfig{}); err == nil {
		t.Fatal("expected nil db to be rejected")
	}
	db := mustDryRun(t)
	if _, err := NewRelay(db, nil, RelayConfig{}); !errors.Is(err, ErrNoPublisher) {
		t.Fatalf("expected ErrNoPublisher, got %v", err)
	}
}

func TestRelayClaimKeepsLeaseFencingSemantics(t *testing.T) {
	db, rec := dryRunDB(t)
	relay, err := NewRelay(db, &fakePublisher{}, RelayConfig{})
	require.NoError(t, err)

	// DryRun cannot materialize rows for the raw CTE, so claim surfaces
	// ErrDryRunModeUnsupported after the statement has been built and logged.
	if _, err := relay.claim(context.Background()); !errors.Is(err, gorm.ErrDryRunModeUnsupported) {
		t.Fatalf("expected DryRun error, got %v", err)
	}

	sql := rec.last()
	for _, needle := range []string{"outbox_events", "FOR UPDATE SKIP LOCKED", "LIMIT", "status = 'publishing'"} {
		if !strings.Contains(sql, needle) {
			t.Fatalf("claim query does not contain %q: %s", needle, sql)
		}
	}
}

type fakePublisher struct {
	calls int
}

func (f *fakePublisher) SendMessageWithRetry(_, _ string, _ []byte, _ int) error {
	f.calls++
	return nil
}

var _ Publisher = (*fakePublisher)(nil)
