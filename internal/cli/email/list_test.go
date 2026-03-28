package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveListPagination(t *testing.T) {
	tests := []struct {
		name         string
		limit        int
		all          bool
		maxItems     int
		wantLimit    int
		wantMaxItems int
	}{
		{
			name:         "single page keeps limit and uses direct fetch sentinel",
			limit:        25,
			wantLimit:    25,
			wantMaxItems: -1,
		},
		{
			name:         "all fetches unlimited pages",
			limit:        25,
			all:          true,
			wantLimit:    200,
			wantMaxItems: 0,
		},
		{
			name:         "all with cap preserves cap",
			limit:        25,
			all:          true,
			maxItems:     500,
			wantLimit:    200,
			wantMaxItems: 500,
		},
		{
			name:         "large limit auto paginates with cap",
			limit:        350,
			wantLimit:    200,
			wantMaxItems: 350,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotMaxItems := resolveListPagination(tt.limit, tt.all, tt.maxItems)
			assert.Equal(t, tt.wantLimit, gotLimit)
			assert.Equal(t, tt.wantMaxItems, gotMaxItems)
		})
	}
}

type testListClient struct {
	*nylas.MockClient
	getMessagesWithCursorFunc func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error)
}

func (c *testListClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	if c.getMessagesWithCursorFunc != nil {
		return c.getMessagesWithCursorFunc(ctx, grantID, params)
	}
	return c.MockClient.GetMessagesWithCursor(ctx, grantID, params)
}

func TestFetchListMessages(t *testing.T) {
	t.Run("applies explicit folder resolution and filters", func(t *testing.T) {
		cmd := newListCmd()
		err := cmd.Flags().Set("unread", "true")
		require.NoError(t, err)
		err = cmd.Flags().Set("starred", "true")
		require.NoError(t, err)

		client := &testListClient{
			MockClient: nylas.NewMockClient(),
		}
		client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "folder-123", Name: "Sent Items"},
			}, nil
		}
		client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			require.NotNil(t, params.Unread)
			require.NotNil(t, params.Starred)
			assert.True(t, *params.Unread)
			assert.True(t, *params.Starred)
			assert.Equal(t, []string{"folder-123"}, params.In)
			assert.Equal(t, "boss@example.com", params.From)
			assert.Equal(t, "key:value", params.MetadataPair)
			return []domain.Message{{ID: "msg-1"}}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit:        25,
			unread:       true,
			starred:      true,
			from:         "boss@example.com",
			folder:       "Sent Items",
			metadataPair: "key:value",
		})

		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "msg-1", messages[0].ID)
	})

	t.Run("warns and falls back to INBOX on folder resolution error", func(t *testing.T) {
		cmd := newListCmd()
		var stderr bytes.Buffer
		cmd.SetErr(&stderr)

		client := &testListClient{
			MockClient: nylas.NewMockClient(),
		}
		client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
			return nil, errors.New("folder lookup failed")
		}
		client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			assert.Equal(t, []string{"INBOX"}, params.In)
			return []domain.Message{{ID: "msg-2"}}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit: 15,
		})

		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Contains(t, stderr.String(), "could not resolve INBOX folder")
	})

	t.Run("falls back to literal folder when explicit folder resolution fails", func(t *testing.T) {
		cmd := newListCmd()
		var stderr bytes.Buffer
		cmd.SetErr(&stderr)

		client := &testListClient{
			MockClient: nylas.NewMockClient(),
		}
		client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
			return nil, errors.New("folder lookup failed")
		}
		client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			assert.Equal(t, []string{"SENT"}, params.In)
			return []domain.Message{{ID: "msg-3"}}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit:  15,
			folder: "SENT",
		})

		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Contains(t, stderr.String(), "could not resolve folder 'SENT'")
	})

	t.Run("leaves folder filter unset when all folders requested", func(t *testing.T) {
		cmd := newListCmd()
		client := &testListClient{
			MockClient: nylas.NewMockClient(),
		}
		client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			assert.Nil(t, params.In)
			return []domain.Message{{ID: "msg-4"}}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit:      15,
			allFolders: true,
		})

		require.NoError(t, err)
		assert.Len(t, messages, 1)
	})

	t.Run("uses literal folder when no match found", func(t *testing.T) {
		cmd := newListCmd()
		client := &testListClient{
			MockClient: nylas.NewMockClient(),
		}
		client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "folder-1", Name: "Archive"},
			}, nil
		}
		client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			assert.Equal(t, []string{"CustomFolder"}, params.In)
			return []domain.Message{{ID: "msg-5"}}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit:  15,
			folder: "CustomFolder",
		})

		require.NoError(t, err)
		assert.Len(t, messages, 1)
	})

	t.Run("uses cursor pagination for all mode", func(t *testing.T) {
		cmd := newListCmd()
		client := &testListClient{
			MockClient: nylas.NewMockClient(),
			getMessagesWithCursorFunc: func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
				switch params.PageToken {
				case "":
					return &domain.MessageListResponse{
						Data: []domain.Message{{ID: "msg-1"}},
						Pagination: domain.Pagination{
							NextCursor: "next",
						},
					}, nil
				case "next":
					return &domain.MessageListResponse{
						Data: []domain.Message{{ID: "msg-2"}},
					}, nil
				default:
					return nil, errors.New("unexpected token")
				}
			},
		}
		client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
			return []domain.Folder{
				{ID: "inbox-id", Name: "Inbox"},
			}, nil
		}

		messages, err := fetchListMessages(context.Background(), cmd, client, "grant-123", listOptions{
			limit: 10,
			all:   true,
		})

		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, []string{"msg-1", "msg-2"}, []string{messages[0].ID, messages[1].ID})
	})
}

func TestRunListStructured(t *testing.T) {
	cmd := newListCmd()
	cmd.Flags().Bool("json", false, "Output in JSON format")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Flags().Set("json", "true")
	require.NoError(t, err)
	err = cmd.Flags().Set("unread", "true")
	require.NoError(t, err)

	client := &testListClient{
		MockClient: nylas.NewMockClient(),
	}
	client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
		return []domain.Folder{{ID: "inbox-id", Name: "Inbox"}}, nil
	}
	client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
		return []domain.Message{{ID: "msg-structured"}}, nil
	}

	resultErr := writeListStructured(context.Background(), cmd, client, "grant-123", listOptions{
		limit:  10,
		unread: true,
	})
	require.NoError(t, resultErr)

	var decoded []domain.Message
	err = json.Unmarshal(stdout.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	assert.Equal(t, "msg-structured", decoded[0].ID)
}
