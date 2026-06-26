package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeNotetakerClient struct {
	ports.NotetakerClient

	err error

	listResult   []domain.Notetaker
	getResult    *domain.Notetaker
	createResult *domain.Notetaker
	updateResult *domain.Notetaker
	mediaResult  *domain.MediaData

	method       string
	grantID      string
	notetakerID  string
	listParams   *domain.NotetakerQueryParams
	createReq    *domain.CreateNotetakerRequest
	updateReq    *domain.UpdateNotetakerRequest
	deleteCalled bool
	leaveCalled  bool
}

func (f *fakeNotetakerClient) ListNotetakers(ctx context.Context, grantID string, params *domain.NotetakerQueryParams) ([]domain.Notetaker, error) {
	f.method = "list"
	f.grantID = grantID
	f.listParams = params
	return f.listResult, f.err
}

func (f *fakeNotetakerClient) GetNotetaker(ctx context.Context, grantID, notetakerID string) (*domain.Notetaker, error) {
	f.method = "get"
	f.grantID = grantID
	f.notetakerID = notetakerID
	return f.getResult, f.err
}

func (f *fakeNotetakerClient) CreateNotetaker(ctx context.Context, grantID string, req *domain.CreateNotetakerRequest) (*domain.Notetaker, error) {
	f.method = "create"
	f.grantID = grantID
	f.createReq = req
	return f.createResult, f.err
}

func (f *fakeNotetakerClient) UpdateNotetaker(ctx context.Context, grantID, notetakerID string, req *domain.UpdateNotetakerRequest) (*domain.Notetaker, error) {
	f.method = "update"
	f.grantID = grantID
	f.notetakerID = notetakerID
	f.updateReq = req
	return f.updateResult, f.err
}

func (f *fakeNotetakerClient) DeleteNotetaker(ctx context.Context, grantID, notetakerID string) error {
	f.method = "delete"
	f.grantID = grantID
	f.notetakerID = notetakerID
	f.deleteCalled = true
	return f.err
}

func (f *fakeNotetakerClient) LeaveNotetaker(ctx context.Context, grantID, notetakerID string) error {
	f.method = "leave"
	f.grantID = grantID
	f.notetakerID = notetakerID
	f.leaveCalled = true
	return f.err
}

func (f *fakeNotetakerClient) GetNotetakerMedia(ctx context.Context, grantID, notetakerID string) (*domain.MediaData, error) {
	f.method = "media"
	f.grantID = grantID
	f.notetakerID = notetakerID
	return f.mediaResult, f.err
}

func TestRegisterNotetakerHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")
	enabled := true

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeNotetakerClient
		assert       func(*testing.T, *fakeNotetakerClient, rpcTestResponse)
	}{
		{
			name:         "notetaker.list returns notetakers",
			method:       "notetaker.list",
			params:       `{"limit":2,"page_token":"cursor-1","state":"complete"}`,
			defaultGrant: "default-grant",
			client: &fakeNotetakerClient{
				listResult: []domain.Notetaker{{ID: "nt-1", State: domain.NotetakerStateComplete}},
			},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "default-grant" {
					t.Fatalf("grantID = %q, want default-grant", client.grantID)
				}
				if client.listParams == nil || client.listParams.Limit != 2 || client.listParams.PageToken != "cursor-1" || client.listParams.State != "complete" {
					t.Fatalf("listParams = %#v, want forwarded query params", client.listParams)
				}

				var result notetakerListResult
				unmarshalResult(t, resp, &result)
				if len(result.Notetakers) != 1 || result.Notetakers[0].ID != "nt-1" {
					t.Fatalf("notetakers = %#v, want nt-1", result.Notetakers)
				}
			},
		},
		{
			name:   "notetaker.list missing grant returns invalid params",
			method: "notetaker.list",
			params: `{}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.method != "" {
					t.Fatalf("method = %q, want no client call", client.method)
				}
			},
		},
		{
			name:         "notetaker.list client error maps to internal error",
			method:       "notetaker.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "notetaker.get returns notetaker",
			method: "notetaker.get",
			params: `{"grant_id":"grant-1","notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{getResult: &domain.Notetaker{ID: "nt-1", State: domain.NotetakerStateAttending}},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "grant-1" || client.notetakerID != "nt-1" {
					t.Fatalf("args = %q/%q, want grant-1/nt-1", client.grantID, client.notetakerID)
				}

				var nt domain.Notetaker
				unmarshalResult(t, resp, &nt)
				if nt.ID != "nt-1" || nt.State != domain.NotetakerStateAttending {
					t.Fatalf("notetaker = %#v, want nt-1 attending", nt)
				}
			},
		},
		{
			name:   "notetaker.get missing grant returns invalid params",
			method: "notetaker.get",
			params: `{"notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "notetaker.get missing notetaker_id returns invalid params",
			method:       "notetaker.get",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.method != "" {
					t.Fatalf("method = %q, want no client call", client.method)
				}
			},
		},
		{
			name:         "notetaker.get client error maps to internal error",
			method:       "notetaker.get",
			params:       `{"notetaker_id":"nt-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:         "notetaker.create forwards request",
			method:       "notetaker.create",
			params:       `{"meeting_link":"https://meet.example/abc","join_time":1710000000,"bot_config":{"name":"Nyla"}}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{createResult: &domain.Notetaker{ID: "nt-1", MeetingLink: "https://meet.example/abc"}},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "default-grant" {
					t.Fatalf("grantID = %q, want default-grant", client.grantID)
				}
				if client.createReq == nil || client.createReq.MeetingLink != "https://meet.example/abc" || client.createReq.JoinTime != 1710000000 {
					t.Fatalf("createReq = %#v, want meeting link and join time", client.createReq)
				}
				if client.createReq.BotConfig == nil || client.createReq.BotConfig.Name != "Nyla" {
					t.Fatalf("BotConfig = %#v, want Nyla", client.createReq.BotConfig)
				}

				var nt domain.Notetaker
				unmarshalResult(t, resp, &nt)
				if nt.ID != "nt-1" {
					t.Fatalf("notetaker = %#v, want nt-1", nt)
				}
			},
		},
		{
			name:   "notetaker.create missing grant returns invalid params",
			method: "notetaker.create",
			params: `{"meeting_link":"https://meet.example/abc"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.createReq != nil {
					t.Fatalf("createReq = %#v, want nil", client.createReq)
				}
			},
		},
		{
			name:         "notetaker.create client error maps to internal error",
			method:       "notetaker.create",
			params:       `{"meeting_link":"https://meet.example/abc"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:         "notetaker.update forwards request",
			method:       "notetaker.update",
			params:       `{"notetaker_id":"nt-1","join_time":1710000100,"name":"Updated","meeting_settings":{"audio_recording":true}}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{updateResult: &domain.Notetaker{ID: "nt-1", State: domain.NotetakerStateScheduled}},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "default-grant" || client.notetakerID != "nt-1" {
					t.Fatalf("args = %q/%q, want default-grant/nt-1", client.grantID, client.notetakerID)
				}
				if client.updateReq == nil || client.updateReq.JoinTime != 1710000100 || client.updateReq.Name != "Updated" {
					t.Fatalf("updateReq = %#v, want join time and name", client.updateReq)
				}
				if client.updateReq.MeetingSettings == nil || client.updateReq.MeetingSettings.AudioRecording == nil || *client.updateReq.MeetingSettings.AudioRecording != enabled {
					t.Fatalf("MeetingSettings = %#v, want audio recording true", client.updateReq.MeetingSettings)
				}
			},
		},
		{
			name:   "notetaker.update missing grant returns invalid params",
			method: "notetaker.update",
			params: `{"notetaker_id":"nt-1","name":"Updated"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.updateReq != nil {
					t.Fatalf("updateReq = %#v, want nil", client.updateReq)
				}
			},
		},
		{
			name:         "notetaker.update missing notetaker_id returns invalid params",
			method:       "notetaker.update",
			params:       `{"grant_id":"grant-1","name":"Updated"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.updateReq != nil {
					t.Fatalf("updateReq = %#v, want nil", client.updateReq)
				}
			},
		},
		{
			name:         "notetaker.update client error maps to internal error",
			method:       "notetaker.update",
			params:       `{"notetaker_id":"nt-1","name":"Updated"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "notetaker.delete returns deleted",
			method: "notetaker.delete",
			params: `{"grant_id":"grant-1","notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if !client.deleteCalled || client.grantID != "grant-1" || client.notetakerID != "nt-1" {
					t.Fatalf("delete = %v %q/%q, want true grant-1/nt-1", client.deleteCalled, client.grantID, client.notetakerID)
				}
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:   "notetaker.delete missing grant returns invalid params",
			method: "notetaker.delete",
			params: `{"notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.deleteCalled {
					t.Fatal("deleteCalled = true, want false")
				}
			},
		},
		{
			name:         "notetaker.delete missing notetaker_id returns invalid params",
			method:       "notetaker.delete",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.deleteCalled {
					t.Fatal("deleteCalled = true, want false")
				}
			},
		},
		{
			name:         "notetaker.delete client error maps to internal error",
			method:       "notetaker.delete",
			params:       `{"notetaker_id":"nt-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "notetaker.leave returns left",
			method: "notetaker.leave",
			params: `{"grant_id":"grant-1","notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if !client.leaveCalled || client.grantID != "grant-1" || client.notetakerID != "nt-1" {
					t.Fatalf("leave = %v %q/%q, want true grant-1/nt-1", client.leaveCalled, client.grantID, client.notetakerID)
				}
				var result leftResult
				unmarshalResult(t, resp, &result)
				if !result.Left {
					t.Fatal("left = false, want true")
				}
			},
		},
		{
			name:   "notetaker.leave missing grant returns invalid params",
			method: "notetaker.leave",
			params: `{"notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.leaveCalled {
					t.Fatal("leaveCalled = true, want false")
				}
			},
		},
		{
			name:         "notetaker.leave missing notetaker_id returns invalid params",
			method:       "notetaker.leave",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.leaveCalled {
					t.Fatal("leaveCalled = true, want false")
				}
			},
		},
		{
			name:         "notetaker.leave client error maps to internal error",
			method:       "notetaker.leave",
			params:       `{"notetaker_id":"nt-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:   "notetaker.media returns media",
			method: "notetaker.media",
			params: `{"grant_id":"grant-1","notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{
				mediaResult: &domain.MediaData{Recording: &domain.MediaFile{URL: "https://files.example/rec.mp4"}},
			},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "grant-1" || client.notetakerID != "nt-1" {
					t.Fatalf("args = %q/%q, want grant-1/nt-1", client.grantID, client.notetakerID)
				}
				var media domain.MediaData
				unmarshalResult(t, resp, &media)
				if media.Recording == nil || media.Recording.URL != "https://files.example/rec.mp4" {
					t.Fatalf("media = %#v, want recording URL", media)
				}
			},
		},
		{
			name:   "notetaker.media missing grant returns invalid params",
			method: "notetaker.media",
			params: `{"notetaker_id":"nt-1"}`,
			client: &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.method != "" {
					t.Fatalf("method = %q, want no client call", client.method)
				}
			},
		},
		{
			name:         "notetaker.media missing notetaker_id returns invalid params",
			method:       "notetaker.media",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.method != "" {
					t.Fatalf("method = %q, want no client call", client.method)
				}
			},
		},
		{
			name:         "notetaker.media client error maps to internal error",
			method:       "notetaker.media",
			params:       `{"notetaker_id":"nt-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeNotetakerClient{err: clientErr},
			assert: func(t *testing.T, client *fakeNotetakerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterNotetakerHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchEmailRequest(t, d, tt.method, tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}
