package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nylas/cli/internal/domain"
	"github.com/rivo/tview"
)

func (v *MessagesView) toggleStar() {
	meta := v.table.SelectedMeta()
	if meta == nil {
		return
	}

	thread, ok := meta.Data.(*domain.Thread)
	if !ok {
		return
	}

	grantID := v.app.config.GrantID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		newStarred := !thread.Starred
		_, err := v.app.config.Client.UpdateThread(ctx, grantID, thread.ID, &domain.UpdateMessageRequest{
			Starred: &newStarred,
		})
		if err != nil {
			v.app.Flash(FlashError, "Failed to update: %v", err)
			return
		}
		v.app.Flash(FlashInfo, "Thread starred")
		v.app.QueueUpdateDraw(func() {
			v.Load()
		})
	}()
}

func (v *MessagesView) markUnread() {
	meta := v.table.SelectedMeta()
	if meta == nil {
		return
	}

	thread, ok := meta.Data.(*domain.Thread)
	if !ok {
		return
	}

	grantID := v.app.config.GrantID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		unread := true
		_, err := v.app.config.Client.UpdateThread(ctx, grantID, thread.ID, &domain.UpdateMessageRequest{
			Unread: &unread,
		})
		if err != nil {
			v.app.Flash(FlashError, "Failed to update: %v", err)
			return
		}
		v.app.Flash(FlashInfo, "Marked as unread")
		v.app.QueueUpdateDraw(func() {
			v.Load()
		})
	}()
}

func (v *MessagesView) showCompose(mode ComposeMode, replyTo *domain.Message) {
	compose := NewComposeView(v.app, mode, replyTo)

	compose.SetOnSent(func() {
		v.app.PopDetail()
		// Refresh messages to show the sent message (non-blocking)
		v.Load()
	})

	compose.SetOnCancel(func() {
		v.app.PopDetail()
		if v.showingDetail {
			// Go back to message detail view - just set focus
		} else {
			v.app.SetFocus(v.table)
		}
	})

	v.app.PushDetail("compose", compose)
}

func (v *MessagesView) showDownloadDialog() {
	if len(v.attachments) == 0 {
		return
	}

	styles := v.app.styles

	// Create list for attachment selection
	list := tview.NewList()
	list.SetBackgroundColor(styles.BgColor)
	list.SetMainTextColor(styles.FgColor)
	list.SetSecondaryTextColor(styles.InfoColor)
	list.SetSelectedBackgroundColor(styles.FocusColor)
	list.SetSelectedTextColor(styles.BgColor)
	list.SetBorder(true)
	list.SetBorderColor(styles.FocusColor)
	list.SetTitle(" Download Attachment ")
	list.SetTitleColor(styles.TitleFg)

	// Add attachments to list
	for i, attInfo := range v.attachments {
		idx := i
		att := attInfo.Attachment
		msgID := attInfo.MessageID
		sizeStr := formatFileSize(att.Size)
		list.AddItem(
			fmt.Sprintf("%s (%s)", att.Filename, sizeStr),
			att.ContentType,
			rune('1'+i),
			func() {
				v.downloadAttachment(msgID, att.ID, att.Filename, idx+1)
			},
		)
	}

	// Handle Escape to close
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			v.app.PopDetail()
			return nil
		}
		return event
	})

	// Center the list
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(list, 60, 0, true).
			AddItem(nil, 0, 1, false), 0, 2, true).
		AddItem(nil, 0, 1, false)

	v.app.PushDetail("download-dialog", flex)
	v.app.SetFocus(list)
}

func (v *MessagesView) downloadAttachment(messageID, attachmentID, filename string, displayNum int) {
	v.app.Flash(FlashInfo, "Downloading %s...", filename)

	grantID := v.app.config.GrantID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		reader, err := v.app.config.Client.DownloadAttachment(ctx, grantID, messageID, attachmentID)
		if err != nil {
			v.app.Flash(FlashError, "Download failed: %v", err)
			return
		}
		defer func() { _ = reader.Close() }()

		// Get Downloads directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			v.app.Flash(FlashError, "Cannot find home directory: %v", err)
			return
		}
		downloadDir := filepath.Join(homeDir, "Downloads")

		// Ensure download directory exists
		if err := os.MkdirAll(downloadDir, 0750); err != nil {
			v.app.Flash(FlashError, "Cannot create Downloads directory: %v", err)
			return
		}

		// Create file with unique name if exists
		destPath, err := safeAttachmentDownloadPath(downloadDir, filename)
		if err != nil {
			v.app.Flash(FlashError, "Cannot create file: %v", err)
			return
		}
		// #nosec G304 -- destPath is sanitized and constrained to downloadDir by safeAttachmentDownloadPath.
		file, destPath, err := v.createUniqueAttachmentFile(destPath)
		if err != nil {
			v.app.Flash(FlashError, "Cannot create file: %v", err)
			return
		}
		defer func() { _ = file.Close() }()

		// Copy content
		written, err := io.Copy(file, reader)
		if err != nil {
			v.app.Flash(FlashError, "Download failed: %v", err)
			return
		}

		v.app.QueueUpdateDraw(func() {
			v.app.PopDetail() // Close download dialog
			v.app.Flash(FlashInfo, "Downloaded %s (%s) to %s", filename, formatFileSize(written), destPath)
		})
	}()
}

func safeAttachmentDownloadPath(downloadDir, filename string) (string, error) {
	safeName := filepath.Base(filename)
	if safeName == "." || safeName == ".." || safeName == string(filepath.Separator) {
		safeName = "attachment"
	}

	// filepath.Abs cleans traversal but intentionally does not resolve symlinked download directories.
	downloadDirAbs, err := filepath.Abs(downloadDir)
	if err != nil {
		return "", fmt.Errorf("resolve download directory: %w", err)
	}

	destPath := filepath.Join(downloadDirAbs, safeName)
	destPathAbs, err := filepath.Abs(destPath)
	if err != nil {
		return "", fmt.Errorf("resolve attachment path: %w", err)
	}

	rel, err := filepath.Rel(downloadDirAbs, destPathAbs)
	if err != nil {
		return "", fmt.Errorf("validate attachment path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("attachment filename %q resolves outside Downloads", filename)
	}

	return destPathAbs, nil
}

func (v *MessagesView) getUniqueFilename(path string) string {
	// Caller must pass a path already sanitized and constrained to the download directory.
	for i := 0; i < 1000; i++ {
		candidate := uniqueAttachmentFilename(path, i)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}

	// Fallback: append timestamp
	return timestampAttachmentFilename(path)
}

func (v *MessagesView) createUniqueAttachmentFile(path string) (*os.File, string, error) {
	// Caller must pass a path already sanitized and constrained to the download directory.
	for i := 0; i < 1000; i++ {
		candidate := uniqueAttachmentFilename(path, i)
		// #nosec G304 -- candidate is derived from a path constrained by safeAttachmentDownloadPath.
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			return file, candidate, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return nil, "", err
	}

	fallback := timestampAttachmentFilename(path)
	// #nosec G304 -- fallback is derived from a path constrained by safeAttachmentDownloadPath.
	file, err := os.OpenFile(fallback, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, "", err
	}
	return file, fallback, nil
}

func uniqueAttachmentFilename(path string, index int) string {
	if index == 0 {
		return path
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(filepath.Base(path), ext)
	return filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, index, ext))
}

func timestampAttachmentFilename(path string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	name := strings.TrimSuffix(filepath.Base(path), ext)
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
}
