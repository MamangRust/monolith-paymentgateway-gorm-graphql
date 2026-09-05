package outbox

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrInvalidInboxKey = errors.New("invalid consumer inbox key")

// ConsumerInbox is the durable deduplication contract used by Kafka handlers.
type ConsumerInbox interface {
	Reserve(ctx context.Context, consumerName, eventKey, topic string, partition int32, offset int64) (bool, bool, int64, error)
	MarkProcessed(ctx context.Context, consumerName, eventKey string, reservationVersion int64) error
	Release(ctx context.Context, consumerName, eventKey string, reservationVersion int64, processingErr error) error
}

// reserveConsumerInboxSQL mirrors the original ReserveConsumerInbox query: an
// INSERT ... ON CONFLICT upsert that only reclaims an expired lease, plus an
// EXISTS check for an already-processed event. Lease fencing prevents two
// consumers from processing the same event concurrently.
const reserveConsumerInboxSQL = `
WITH reserved AS (
    INSERT INTO consumer_inbox (
        consumer_name, event_key, topic, partition_id, message_offset,
        status, attempts, reservation_version, lease_until, last_error, processed_at
    )
    VALUES (?, ?, ?, ?, ?, 'processing', 1, 1,
            current_timestamp + interval '1 minute', '', NULL)
    ON CONFLICT (consumer_name, event_key) DO UPDATE
    SET status = 'processing',
        attempts = consumer_inbox.attempts + 1,
        reservation_version = consumer_inbox.reservation_version + 1,
        lease_until = current_timestamp + interval '1 minute',
        last_error = '',
        topic = EXCLUDED.topic,
        partition_id = EXCLUDED.partition_id,
        message_offset = EXCLUDED.message_offset
    WHERE consumer_inbox.status <> 'processed'
      AND consumer_inbox.lease_until <= current_timestamp
    RETURNING reservation_version
)
SELECT
    EXISTS (SELECT 1 FROM reserved) AS reserved,
    EXISTS (
        SELECT 1
        FROM consumer_inbox ci
        WHERE ci.consumer_name = ?
          AND ci.event_key = ?
          AND ci.status = 'processed'
    ) AS processed,
    COALESCE(
        (SELECT reservation_version FROM reserved),
        (SELECT ci.reservation_version FROM consumer_inbox ci WHERE ci.consumer_name = ? AND ci.event_key = ?)
    )::BIGINT AS reservation_version`

// markConsumerInboxProcessedSQL completes only the active reservation.
const markConsumerInboxProcessedSQL = `
UPDATE consumer_inbox
SET status = 'processed', processed_at = current_timestamp,
    lease_until = current_timestamp, last_error = ''
WHERE consumer_name = ? AND event_key = ?
  AND status = 'processing' AND reservation_version = ?`

// releaseConsumerInboxSQL releases only the active reservation.
const releaseConsumerInboxSQL = `
UPDATE consumer_inbox
SET status = 'pending', lease_until = current_timestamp,
    last_error = ?
WHERE consumer_name = ? AND event_key = ?
  AND status = 'processing' AND reservation_version = ?`

// inboxReservationRow mirrors the SELECT columns of reserveConsumerInboxSQL.
type inboxReservationRow struct {
	Reserved           bool
	Processed          bool
	ReservationVersion int64
}

// Reserve claims an event for a consumer. It returns false when the event was
// already processed. An expired processing lease may be reclaimed after a
// consumer crashes.
func Reserve(ctx context.Context, db *gorm.DB, consumerName, eventKey, topic string, partition int32, offset int64) (bool, bool, int64, error) {
	if db == nil || consumerName == "" || eventKey == "" {
		return false, false, 0, ErrInvalidInboxKey
	}
	var row inboxReservationRow
	if err := db.WithContext(ctx).Raw(reserveConsumerInboxSQL,
		consumerName, eventKey, topic, partition, offset,
		consumerName, eventKey,
		consumerName, eventKey,
	).Scan(&row).Error; err != nil {
		return false, false, 0, err
	}
	return row.Reserved, row.Processed, row.ReservationVersion, nil
}

func MarkProcessed(ctx context.Context, db *gorm.DB, consumerName, eventKey string, reservationVersion int64) error {
	if db == nil || consumerName == "" || eventKey == "" {
		return ErrInvalidInboxKey
	}
	return db.WithContext(ctx).Exec(markConsumerInboxProcessedSQL,
		consumerName, eventKey, reservationVersion,
	).Error
}

func Release(ctx context.Context, db *gorm.DB, consumerName, eventKey string, reservationVersion int64, processingErr error) error {
	if db == nil || consumerName == "" || eventKey == "" {
		return ErrInvalidInboxKey
	}
	lastError := "consumer processing failed"
	if processingErr != nil {
		lastError = processingErr.Error()
	}
	return db.WithContext(ctx).Exec(releaseConsumerInboxSQL,
		lastError, consumerName, eventKey, reservationVersion,
	).Error
}

// PostgresInbox adapts a *gorm.DB to ConsumerInbox. Reservation and completion
// are committed independently because an external side effect cannot share a
// PostgreSQL transaction with the Kafka consumer.
type PostgresInbox struct {
	db *gorm.DB
}

func NewPostgresInbox(db *gorm.DB) (*PostgresInbox, error) {
	if db == nil {
		return nil, errors.New("inbox db is nil")
	}
	return &PostgresInbox{db: db}, nil
}

func (i *PostgresInbox) Reserve(ctx context.Context, consumerName, eventKey, topic string, partition int32, offset int64) (bool, bool, int64, error) {
	var (
		reserved, processed bool
		version             int64
		err                 error
	)
	txErr := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reserved, processed, version, err = Reserve(ctx, tx, consumerName, eventKey, topic, partition, offset)
		return err
	})
	if txErr != nil {
		return false, false, 0, txErr
	}
	return reserved, processed, version, nil
}

func (i *PostgresInbox) MarkProcessed(ctx context.Context, consumerName, eventKey string, reservationVersion int64) error {
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return MarkProcessed(ctx, tx, consumerName, eventKey, reservationVersion)
	})
}

func (i *PostgresInbox) Release(ctx context.Context, consumerName, eventKey string, reservationVersion int64, processingErr error) error {
	return i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return Release(ctx, tx, consumerName, eventKey, reservationVersion, processingErr)
	})
}
