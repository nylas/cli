package tui

import (
	"testing"
)

func TestTable(t *testing.T) {
	styles := DefaultStyles()
	table := NewTable(styles)

	if table == nil {
		t.Fatal("NewTable() returned nil")
	}

	// Set columns
	columns := []Column{
		{Title: "ID", Width: 10},
		{Title: "Name", Expand: true},
		{Title: "Status", Width: 8},
	}
	table.SetColumns(columns)

	// Set data
	data := [][]string{
		{"1", "Item 1", "Active"},
		{"2", "Item 2", "Inactive"},
		{"3", "Item 3", "Pending"},
	}
	meta := []RowMeta{
		{ID: "1", Unread: true},
		{ID: "2", Starred: true},
		{ID: "3", Error: true},
	}
	table.SetData(data, meta)

	// Verify row count
	if table.GetRowCount() != 3 {
		t.Errorf("GetRowCount() = %d, want 3", table.GetRowCount())
	}
}

func TestTableSelection(t *testing.T) {
	styles := DefaultStyles()
	table := NewTable(styles)

	// Set up simple data
	table.SetColumns([]Column{
		{Title: "Name", Expand: true},
	})
	table.SetData(
		[][]string{{"Item 1"}, {"Item 2"}, {"Item 3"}},
		[]RowMeta{{ID: "1"}, {ID: "2"}, {ID: "3"}},
	)

	// Test initial selection (should be first data row)
	row := table.GetSelectedRow()
	if row != 0 {
		t.Errorf("Initial GetSelectedRow() = %d, want 0", row)
	}

	// Test SelectedMeta
	meta := table.SelectedMeta()
	if meta == nil {
		t.Fatal("SelectedMeta() returned nil")
		return
	}
	if meta.ID != "1" {
		t.Errorf("SelectedMeta().ID = %q, want %q", meta.ID, "1")
	}
}

func TestTable_SelectedMeta(t *testing.T) {
	styles := DefaultStyles()
	table := NewTable(styles)

	table.SetColumns([]Column{{Title: "Name", Expand: true}})
	table.SetData(
		[][]string{{"Item 1"}, {"Item 2"}},
		[]RowMeta{{ID: "id-1"}, {ID: "id-2"}},
	)

	// Select first row (row 1, since row 0 is header)
	table.Select(1, 0)

	meta := table.SelectedMeta()
	if meta == nil {
		t.Fatal("SelectedMeta() returned nil")
		return
	}
	if meta.ID != "id-1" {
		t.Errorf("SelectedMeta().ID = %q, want %q", meta.ID, "id-1")
	}
}

func TestTable_SetData(t *testing.T) {
	styles := DefaultStyles()
	table := NewTable(styles)

	table.SetColumns([]Column{{Title: "Name", Expand: true}})

	// Verify initial state
	initialCount := table.GetRowCount()

	// Set data
	table.SetData(
		[][]string{{"Item 1"}, {"Item 2"}},
		[]RowMeta{{ID: "id-1"}, {ID: "id-2"}},
	)

	// Verify data was set
	afterCount := table.GetRowCount()
	if afterCount <= initialCount {
		t.Errorf("After SetData(), GetRowCount() = %d, should be > %d", afterCount, initialCount)
	}
}

func TestTable_GetRowCount(t *testing.T) {
	styles := DefaultStyles()
	table := NewTable(styles)

	table.SetColumns([]Column{{Title: "Name", Expand: true}})
	table.SetData(
		[][]string{{"Item 1"}, {"Item 2"}, {"Item 3"}},
		[]RowMeta{{ID: "id-1"}, {ID: "id-2"}, {ID: "id-3"}},
	)

	// GetRowCount should return the count including header
	count := table.GetRowCount()
	if count < 3 {
		t.Errorf("GetRowCount() = %d, want >= 3", count)
	}
}
