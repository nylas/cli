package contacts

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testContactsClient struct {
	*nylas.MockClient
	getContactsWithCursorFunc func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error)
}

func (c *testContactsClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	if c.getContactsWithCursorFunc != nil {
		return c.getContactsWithCursorFunc(ctx, grantID, params)
	}
	return c.MockClient.GetContactsWithCursor(ctx, grantID, params)
}

func TestFetchContacts(t *testing.T) {
	t.Run("uses direct fetch when maxItems is zero", func(t *testing.T) {
		client := &testContactsClient{MockClient: nylas.NewMockClient()}
		client.GetContactsFunc = func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
			assert.Equal(t, 25, params.Limit)
			return []domain.Contact{{ID: "contact-1"}}, nil
		}

		contacts, err := fetchContacts(context.Background(), client, "grant-123", &domain.ContactQueryParams{Limit: 25}, 0)

		require.NoError(t, err)
		assert.Len(t, contacts, 1)
		assert.Equal(t, "contact-1", contacts[0].ID)
	})

	t.Run("uses cursor pagination when maxItems is positive", func(t *testing.T) {
		client := &testContactsClient{
			MockClient: nylas.NewMockClient(),
			getContactsWithCursorFunc: func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
				switch params.PageToken {
				case "":
					return &domain.ContactListResponse{
						Data: []domain.Contact{{ID: "contact-1"}, {ID: "contact-2"}},
						Pagination: domain.Pagination{
							NextCursor: "next",
						},
					}, nil
				case "next":
					return &domain.ContactListResponse{
						Data: []domain.Contact{{ID: "contact-3"}},
					}, nil
				default:
					return nil, errors.New("unexpected cursor")
				}
			},
		}

		contacts, err := fetchContacts(context.Background(), client, "grant-123", &domain.ContactQueryParams{Limit: 500}, 3)

		require.NoError(t, err)
		assert.Len(t, contacts, 3)
		assert.Equal(t, []string{"contact-1", "contact-2", "contact-3"}, []string{contacts[0].ID, contacts[1].ID, contacts[2].ID})
	})

	t.Run("returns pagination errors", func(t *testing.T) {
		client := &testContactsClient{
			MockClient: nylas.NewMockClient(),
			getContactsWithCursorFunc: func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
				return nil, errors.New("boom")
			},
		}

		contacts, err := fetchContacts(context.Background(), client, "grant-123", &domain.ContactQueryParams{Limit: 500}, 3)

		require.Error(t, err)
		assert.Nil(t, contacts)
		assert.Contains(t, err.Error(), "failed to fetch page 1")
	})
}
