package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GetFolders returns demo folders.
func (d *DemoClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	return []domain.Folder{
		{ID: "inbox", Name: "Inbox", SystemFolder: "inbox", TotalCount: 1247},
		{ID: "sent", Name: "Sent", SystemFolder: "sent", TotalCount: 532},
		{ID: "drafts", Name: "Drafts", SystemFolder: "drafts", TotalCount: 3},
		{ID: "trash", Name: "Trash", SystemFolder: "trash", TotalCount: 45},
		{ID: "work", Name: "Work", TotalCount: 89},
		{ID: "personal", Name: "Personal", TotalCount: 156},
	}, nil
}

// GetFolder returns a demo folder.
func (d *DemoClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	return &domain.Folder{ID: folderID, Name: "Demo Folder"}, nil
}

// CreateFolder simulates creating a folder.
func (d *DemoClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	return &domain.Folder{ID: "new-folder", Name: req.Name}, nil
}

// UpdateFolder simulates updating a folder.
func (d *DemoClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	return &domain.Folder{ID: folderID, Name: req.Name}, nil
}

// DeleteFolder simulates deleting a folder.
func (d *DemoClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	return nil
}
