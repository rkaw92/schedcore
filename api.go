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
	TenantId string `path:"tenantId"`
	Body     struct {
		TenantId    string          `json:"tenantId"`
		TimerId     string          `json:"timerId"`
		NextAt      time.Time       `json:"nextAt"`
		Schedule    string          `json:"schedule"`
		Payload     json.RawMessage `json:"payload"`
		Destination string          `json:"destination"`
	}
}

type CreateTimerOutput struct {
	Location string `header:"Location"`
	Body     struct {
		TenantId         string    `json:"tenantId"`
		TimerId          string    `json:"timerId"`
		Ushard           int16     `json:"ushard"`
		NextAt           time.Time `json:"nextAt"`
		NextInvocationId string    `json:"nextInvocationId"`
	}
}

type DeleteTimerInput struct {
	TenantId string `path:"tenantId"`
	TimerId  string `path:"timerId"`
}

type DeleteTimerOutput struct {
	Body struct{}
}

func runAPI(adminDb TimerStoreForAdmin, config Config) {
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("schedcore", "0.1.0"))

	huma.Register(api, huma.Operation{
		OperationID: "create-timer",
		Method:      http.MethodPost,
		Path:        "/tenants/{tenantId}/timers",
		Summary:     "Create or update a timer",
		Tags:        []string{"timers"},
	}, func(ctx context.Context, input *CreateTimerInput) (*CreateTimerOutput, error) {
		if input.TenantId != input.Body.TenantId {
			return nil, huma.Error400BadRequest("tenantId in URL must equal tenantId in body")
		}
		tenantId, parseError := uuid.Parse(input.Body.TenantId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("tenantId must be a valid UUID", parseError)
		}
		timerId, parseError := uuid.Parse(input.Body.TimerId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("timerId must be a valid UUID", parseError)
		}
		timer := &Timer{
			TenantId:         tenantId,
			TimerId:          timerId,
			NextAt:           input.Body.NextAt,
			Ushard:           uuid2ushard(timerId, config.TOTAL_USHARDS),
			Schedule:         input.Body.Schedule,
			Payload:          input.Body.Payload,
			Destination:      input.Body.Destination,
			NextInvocationId: GenInvocationId(),
		}
		// Enforce creating timers in advance
		if timer.NextAt.Before(time.Now().Add(MIN_TIME_HORIZON)) {
			timer.NextAt = time.Now().Add(MIN_TIME_HORIZON)
		}
		timer.NextAt = timer.NextAt.Truncate(time.Second)
		// Compute from schedule, if any
		timer.NextAt = timer.Next()
		err := adminDb.Create(timer)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create timer", err)
		}
		resp := &CreateTimerOutput{}
		resp.Body.TenantId = timer.TenantId.String()
		resp.Body.TimerId = timer.TimerId.String()
		resp.Body.Ushard = timer.Ushard
		resp.Body.NextAt = timer.NextAt
		resp.Body.NextInvocationId = timer.NextInvocationId.String()
		resp.Location = "/tenants/" + resp.Body.TenantId + "/timers/" + resp.Body.TimerId

		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-timer",
		Method:      http.MethodDelete,
		Path:        "/tenants/{tenantId}/timers/{timerId}",
		Summary:     "Delete a timer",
		Tags:        []string{"timers"},
	}, func(ctx context.Context, input *DeleteTimerInput) (*DeleteTimerOutput, error) {
		tenantId, parseError := uuid.Parse(input.TenantId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("tenantId must be a valid UUID", parseError)
		}
		timerId, parseError := uuid.Parse(input.TimerId)
		if parseError != nil {
			return nil, huma.Error400BadRequest("timerId must be a valid UUID", parseError)
		}
		ushard := uuid2ushard(timerId, config.TOTAL_USHARDS)
		err := adminDb.Delete(tenantId, timerId, ushard)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to delete timer", err)
		}
		resp := &DeleteTimerOutput{}

		return resp, nil
	})

	// TODO: Custom port
	http.ListenAndServe(":1200", router)
}
