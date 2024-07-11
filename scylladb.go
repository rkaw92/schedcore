package main

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type ScyllaStore struct {
	session *gocql.Session
}

func NewScyllaStore(dbUrl url.URL) (*ScyllaStore, error) {
	hosts := []string{dbUrl.Host}

	cluster := gocql.NewCluster(hosts...)
	keyspace, hasPath := strings.CutPrefix(dbUrl.Path, "/")
	if hasPath {
		cluster.Keyspace = keyspace
	}
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return &ScyllaStore{
		session,
	}, nil
}

func (store *ScyllaStore) Create(timer *Timer) error {
	ctx := context.TODO()
	err := store.session.Query(`INSERT INTO timers (
		tenant_id, timer_id, ushard, next_at, schedule, payload, destination, next_invocation_id
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?
	)`,
		[16]byte(timer.TenantId),
		[16]byte(timer.TimerId),
		timer.Ushard,
		timer.NextAt,
		timer.Schedule,
		// NOTE: We don't set "done=false" on create to preserve idempotence - don't re-create done timers.
		timer.Payload,
		timer.Destination,
		[16]byte(timer.NextInvocationId),
	).WithContext(ctx).Consistency(gocql.LocalQuorum).Exec()
	return err
}

func (store *ScyllaStore) Delete(tenantId uuid.UUID, timerId uuid.UUID, ushard int16) error {
	ctx := context.TODO()
	err := store.session.Query(
		"DELETE FROM timers WHERE tenant_id = ? AND timer_id = ? AND ushard = ?",
		[16]byte(tenantId),
		[16]byte(timerId),
		ushard,
	).WithContext(ctx).Consistency(gocql.LocalQuorum).Exec()
	return err
}

func (store *ScyllaStore) GetPendingTimers(next_at time.Time, ushard int16) ([]Timer, error) {
	all := make([]Timer, 0, 64)
	ctx := context.TODO()

	scanner := store.session.Query(
		"SELECT tenant_id, timer_id, done, schedule, payload, destination, next_invocation_id FROM timers_mat WHERE ushard = ? AND next_at = ?",
		ushard,
		next_at,
	).WithContext(ctx).Consistency(gocql.One).Iter().Scanner()
	for scanner.Next() {
		var (
			tenantId         gocql.UUID
			timerId          gocql.UUID
			done             bool
			schedule         string
			payloadJSON      []byte
			destination      string
			nextInvocationId gocql.UUID
		)
		err := scanner.Scan(&tenantId, &timerId, &done, &schedule, &payloadJSON, &destination, &nextInvocationId)
		if err != nil {
			return nil, err
		}
		var payload interface{}
		err = json.Unmarshal(payloadJSON, &payload)
		if err != nil {
			return nil, err
		}
		all = append(all, Timer{
			next_at,
			uuid.Must(uuid.FromBytes(tenantId.Bytes())),
			uuid.Must(uuid.FromBytes(timerId.Bytes())),
			ushard,
			schedule,
			done,
			payload,
			destination,
			uuid.Must(uuid.FromBytes(nextInvocationId.Bytes())),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return all, nil
}

func (store *ScyllaStore) UpdateTimers(updates []TimerUpdate) error {
	ctx := context.TODO()
	batch := store.session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
	batch.SetConsistency(gocql.LocalQuorum)
	for _, update := range updates {
		batch.Query(
			"UPDATE timers SET next_at = ?, next_invocation_id = ?, done = ? WHERE tenant_id = ? AND timer_id = ? AND ushard = ?",
			update.SetNextAt,
			[16]byte(update.SetNextInvocationId),
			update.IsDone,
			[16]byte(update.Timer.TenantId),
			[16]byte(update.Timer.TimerId),
			update.Timer.Ushard,
		)
	}
	err := store.session.ExecuteBatch(batch)
	return err
}

func (store *ScyllaStore) GetState(myUshard int16) (RunnerState, error) {
	ctx := context.TODO()
	state := RunnerState{MyUshard: myUshard}
	err := store.session.Query(
		"SELECT next FROM runners WHERE ushard = ?",
		myUshard,
	).WithContext(ctx).Consistency(gocql.One).Scan(&state.Next)
	if err != nil && !errors.Is(err, gocql.ErrNotFound) {
		return state, err
	}
	return state, nil
}

func (store *ScyllaStore) SaveState(newState RunnerState) error {
	ctx := context.TODO()
	err := store.session.Query(
		"UPDATE runners SET next = ? WHERE ushard = ?",
		newState.Next,
		newState.MyUshard,
	).WithContext(ctx).Consistency(gocql.LocalOne).Exec()
	return err
}

func (store *ScyllaStore) LogTimerInvocations(timers []*Timer) error {
	ctx := context.TODO()
	batch := store.session.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
	batch.SetConsistency(gocql.LocalOne)
	now := time.Now()
	for _, timer := range timers {
		batch.Query(
			"INSERT INTO invocations (tenant_id, timer_id, invocation_id, scheduled_at, real_at) VALUES (?, ?, ?, ?, ?)",
			[16]byte(timer.TenantId),
			[16]byte(timer.TimerId),
			[16]byte(timer.NextInvocationId),
			timer.NextAt,
			now,
		)
	}
	err := store.session.ExecuteBatch(batch)
	return err
}
