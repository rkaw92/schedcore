package main

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

const MIN_TIME_HORIZON = time.Second * 30

type CreateTimerInput struct {
	Body struct {
		TenantId    string          `json:"tenantId"`
		TimerId     string          `json:"timerId"`
		NextAt      time.Time       `json:"nextAt"`
		Schedule    string          `json:"schedule"`
		Enabled     bool            `json:"enabled" required:"false"`
		Payload     json.RawMessage `json:"payload"`
		Destination string          `json:"destination"`
	}
}

type CreateTimerOutput struct {
	Body struct {
		TenantId string    `json:"tenantId"`
		TimerId  string    `json:"timerId"`
		Ushard   int16     `json:"ushard"`
		NextAt   time.Time `json:"nextAt"`
	}
}

func runAPI(adminDb TimerStoreForAdmin) {
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("schedcore", "0.1.0"))
	huma.Register(api, huma.Operation{
		OperationID: "create-timer",
		Method:      http.MethodPost,
		Path:        "/timers",
		Summary:     "Create or update a timer",
		Tags:        []string{"timers"},
	}, func(ctx context.Context, input *CreateTimerInput) (*CreateTimerOutput, error) {
		tenantId, parseError := uuid.Parse(input.Body.TenantId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("tenantId must be a valid UUID", parseError)
		}
		timerId, parseError := uuid.Parse(input.Body.TimerId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("timerId must be a valid UUID", parseError)
		}
		timer := &Timer{
			TenantId:    tenantId,
			TimerId:     timerId,
			NextAt:      input.Body.NextAt,
			Ushard:      uuid2ushard(timerId),
			Schedule:    input.Body.Schedule,
			Enabled:     true,
			Done:        false,
			Payload:     input.Body.Payload,
			Destination: input.Body.Destination,
		}
		// Enforce creating timers in advance
		if timer.NextAt.Before(time.Now().Add(MIN_TIME_HORIZON)) {
			timer.NextAt = time.Now().Add(MIN_TIME_HORIZON)
		}
		// Compute from schedule, if any
		timer.NextAt = timer.Next()
		err := adminDb.Create(timer)
		if err != nil {
			return nil, err
		}
		resp := &CreateTimerOutput{}
		resp.Body.TenantId = timer.TenantId.String()
		resp.Body.TimerId = timer.TimerId.String()
		resp.Body.Ushard = timer.Ushard
		resp.Body.NextAt = timer.NextAt

		return resp, nil
	})

	// TODO: Custom port
	http.ListenAndServe(":1200", router)
}
