package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	m.GetFoldersCalled = true
	m.LastGrantID = grantID
	if m.GetFoldersFunc != nil {
		return m.GetFoldersFunc(ctx, grantID)
	}
	return []domain.Folder{
		{ID: "inbox", Name: "Inbox", SystemFolder: "inbox"},
		{ID: "sent", Name: "Sent", SystemFolder: "sent"},
		{ID: "drafts", Name: "Drafts", SystemFolder: "drafts"},
	}, nil
}

// GetFolder retrieves a single folder.
func (m *MockClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	m.GetFolderCalled = true
	m.LastGrantID = grantID
	m.LastFolderID = folderID
	if m.GetFolderFunc != nil {
		return m.GetFolderFunc(ctx, grantID, folderID)
	}
	return &domain.Folder{
		ID:      folderID,
		GrantID: grantID,
		Name:    "Test Folder",
	}, nil
}

// CreateFolder creates a new folder.
func (m *MockClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	m.CreateFolderCalled = true
	m.LastGrantID = grantID
	if m.CreateFolderFunc != nil {
		return m.CreateFolderFunc(ctx, grantID, req)
	}
	return &domain.Folder{
		ID:      "new-folder-id",
		GrantID: grantID,
		Name:    req.Name,
	}, nil
}

// UpdateFolder updates an existing folder.
func (m *MockClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	m.UpdateFolderCalled = true
	m.LastGrantID = grantID
	m.LastFolderID = folderID
	if m.UpdateFolderFunc != nil {
		return m.UpdateFolderFunc(ctx, grantID, folderID, req)
	}
	return &domain.Folder{
		ID:      folderID,
		GrantID: grantID,
		Name:    req.Name,
	}, nil
}

// DeleteFolder deletes a folder.
func (m *MockClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	m.DeleteFolderCalled = true
	m.LastGrantID = grantID
	m.LastFolderID = folderID
	if m.DeleteFolderFunc != nil {
		return m.DeleteFolderFunc(ctx, grantID, folderID)
	}
	return nil
}

// ListAttachments retrieves all attachments for a message.
