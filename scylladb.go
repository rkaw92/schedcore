package main

import (
	"context"
	"errors"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type ScyllaStore struct {
	session *gocql.Session
}

func NewScyllaStore(hosts []string, keyspace string) (*ScyllaStore, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return &ScyllaStore{
		session,
	}, nil
}

func (store *ScyllaStore) GetPendingTimers(next_at time.Time, ushard int16) ([]Timer, error) {
	all := make([]Timer, 0, 64)
	ctx := context.TODO()

	scanner := store.session.Query(
		"SELECT tenant_id, timer_id, done, enabled, schedule, payload, destination FROM timers_mat WHERE ushard = ? AND next_at = ?",
		ushard,
		next_at,
	).WithContext(ctx).Consistency(gocql.One).Iter().Scanner()
	for scanner.Next() {
		var (
			tenantId    gocql.UUID
			timerId     gocql.UUID
			done        bool
			enabled     bool
			schedule    string
			payloadJSON []byte
			destination string
		)
		err := scanner.Scan(&tenantId, &timerId, &done, &enabled, &schedule, &payloadJSON, &destination)
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
			enabled,
			done,
			payload,
			destination,
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
			"UPDATE timers SET next_at = ?, done = ? WHERE tenant_id = ? AND timer_id = ? AND ushard = ?",
			update.SetNextAt,
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
			"INSERT INTO invocations (tenant_id, timer_id, scheduled_at, real_at) VALUES (?, ?, ?, ?)",
			[16]byte(timer.TenantId),
			[16]byte(timer.TimerId),
			timer.NextAt,
			now,
		)
	}
	err := store.session.ExecuteBatch(batch)
	return err
}
