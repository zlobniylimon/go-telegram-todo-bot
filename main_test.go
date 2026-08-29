package main

import (
	"sync"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
)

func TestButtonText_Enabled(t *testing.T) {
	item := Item{Name: "Milk", Enable: true}
	got := buttonText(item)
	want := "✅ Milk"
	if got != want {
		t.Errorf("buttonText(%+v) = %q, want %q", item, got, want)
	}
}

func TestButtonText_Disabled(t *testing.T) {
	item := Item{Name: "Bread", Enable: false}
	got := buttonText(item)
	want := "❌ Bread"
	if got != want {
		t.Errorf("buttonText(%+v) = %q, want %q", item, got, want)
	}
}

func TestLockedImage_Locked(t *testing.T) {
	data := &ChatListData{Locked: true}
	got := lockedImage(data)
	want := "🔒"
	if got != want {
		t.Errorf("lockedImage(%+v) = %q, want %q", data, got, want)
	}
}

func TestLockedImage_Unlocked(t *testing.T) {
	data := &ChatListData{Locked: false}
	got := lockedImage(data)
	want := "🔓"
	if got != want {
		t.Errorf("lockedImage(%+v) = %q, want %q", data, got, want)
	}
}

func TestParseShoppingList_Empty(t *testing.T) {
	data := &ChatListData{}
	parseShoppingList(data, "")
	if len(data.Items) != 1 || data.Items[0].Name != "" {
		t.Errorf("parseShoppingList(empty, \"\") = %+v, want single empty item", data.Items)
	}
}

func TestParseShoppingList_SingleItem(t *testing.T) {
	data := &ChatListData{}
	parseShoppingList(data, "Milk")
	if len(data.Items) != 1 {
		t.Fatalf("parseShoppingList(empty, \"Milk\") = %+v, want 1 item", data.Items)
	}
	if data.Items[0].Name != "Milk" || data.Items[0].Enable != false || data.Items[0].ID != 1 {
		t.Errorf("expected item Milk/disabled/id=1, got %+v", data.Items[0])
	}
	if data.NextItemID != 1 {
		t.Errorf("expected NextItemID 1, got %d", data.NextItemID)
	}
}

func TestParseShoppingList_MultipleItems(t *testing.T) {
	data := &ChatListData{}
	parseShoppingList(data, "Milk\nBread\nEggs")
	if len(data.Items) != 3 {
		t.Fatalf("expected 3 items, got %d: %+v", len(data.Items), data.Items)
	}
	expected := []string{"Milk", "Bread", "Eggs"}
	for i, name := range expected {
		if data.Items[i].Name != name || data.Items[i].Enable != false || data.Items[i].ID != i+1 {
			t.Errorf("item %d: expected %s/disabled/id=%d, got %+v", i, name, i+1, data.Items[i])
		}
	}
}

func TestParseShoppingList_AppendsToList(t *testing.T) {
	data := &ChatListData{
		Items:      []Item{{ID: 1, Name: "Butter", Enable: true}},
		NextItemID: 1,
	}
	parseShoppingList(data, "Milk")
	if len(data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d: %+v", len(data.Items), data.Items)
	}
	if data.Items[0].Name != "Butter" || data.Items[0].Enable != true || data.Items[0].ID != 1 {
		t.Errorf("first item should be preserved Butter/enabled, got %+v", data.Items[0])
	}
	if data.Items[1].Name != "Milk" || data.Items[1].Enable != false || data.Items[1].ID != 2 {
		t.Errorf("second item should be Milk/disabled/id=2, got %+v", data.Items[1])
	}
}

func TestGenerateChatKey(t *testing.T) {
	msg := &models.Message{
		Chat:            models.Chat{ID: 12345},
		MessageThreadID: 678,
	}
	got := generateChatKey(msg)
	want := "12345:678"
	if got != want {
		t.Errorf("generateChatKey(%+v) = %q, want %q", msg, got, want)
	}
}

func TestGenerateChatKey_ZeroThreadID(t *testing.T) {
	msg := &models.Message{
		Chat: models.Chat{ID: 42},
	}
	got := generateChatKey(msg)
	want := "42:0"
	if got != want {
		t.Errorf("generateChatKey(%+v) = %q, want %q", msg, got, want)
	}
}

func TestFormatListDataButton_EmptyList(t *testing.T) {
	data := &ChatListData{Locked: false}
	kb := formatListDataButton(data)
	markup, ok := kb.(*models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("expected *models.InlineKeyboardMarkup, got %T", kb)
	}

	if len(markup.InlineKeyboard) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(markup.InlineKeyboard))
	}

	deleteRow := markup.InlineKeyboard[0]
	if len(deleteRow) != 2 {
		t.Fatalf("expected 2 buttons in delete row, got %d", len(deleteRow))
	}
	if deleteRow[0].CallbackData != "btn_empty_list" {
		t.Errorf("expected btn_empty_list, got %s", deleteRow[0].CallbackData)
	}
	if deleteRow[1].CallbackData != "btn_refresh_list" {
		t.Errorf("expected btn_refresh_list, got %s", deleteRow[1].CallbackData)
	}

	lockRow := markup.InlineKeyboard[1]
	if len(lockRow) != 1 {
		t.Fatalf("expected 1 button in lock row, got %d", len(lockRow))
	}
	if lockRow[0].CallbackData != "btn_list_locked" {
		t.Errorf("expected btn_list_locked, got %s", lockRow[0].CallbackData)
	}
	if lockRow[0].Text != "🔓" {
		t.Errorf("expected unlocked icon, got %s", lockRow[0].Text)
	}
}

func TestFormatListDataButton_WithItems(t *testing.T) {
	data := &ChatListData{
		Items:  []Item{{ID: 1, Name: "Milk", Enable: false}, {ID: 2, Name: "Bread", Enable: true}},
		Locked: true,
	}
	kb := formatListDataButton(data)
	markup, ok := kb.(*models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("expected *models.InlineKeyboardMarkup, got %T", kb)
	}

	if len(markup.InlineKeyboard) < 4 {
		t.Fatalf("expected at least 4 rows (2 items + 2 control), got %d", len(markup.InlineKeyboard))
	}

	item0 := markup.InlineKeyboard[0][0]
	if item0.Text != "❌ Milk" || item0.CallbackData != "btn_item_1" {
		t.Errorf("first item: got %s / %s", item0.Text, item0.CallbackData)
	}

	item1 := markup.InlineKeyboard[1][0]
	if item1.Text != "✅ Bread" || item1.CallbackData != "btn_item_2" {
		t.Errorf("second item: got %s / %s", item1.Text, item1.CallbackData)
	}

	lockRow := markup.InlineKeyboard[len(markup.InlineKeyboard)-1][0]
	if lockRow.Text != "🔒" {
		t.Errorf("expected locked icon, got %s", lockRow.Text)
	}
}

func TestApplyCallback_ToggleItem(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 1, Name: "Milk", Enable: false}}}
	applyCallback(data, "btn_item_1")
	if !data.Items[0].Enable {
		t.Errorf("expected item toggled enabled, got %+v", data.Items[0])
	}
	applyCallback(data, "btn_item_1")
	if data.Items[0].Enable {
		t.Errorf("expected item toggled back to disabled, got %+v", data.Items[0])
	}
}

func TestApplyCallback_ToggleMissingItem_NoPanic(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 1, Name: "Milk", Enable: false}}}
	applyCallback(data, "btn_item_99")
	if len(data.Items) != 1 || data.Items[0].Enable {
		t.Errorf("stale callback should be a no-op, got %+v", data.Items)
	}
}

func TestApplyCallback_ToggleByStableID(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 3, Name: "Milk", Enable: false}, {ID: 8, Name: "Bread", Enable: false}}}
	applyCallback(data, "btn_item_3")
	if !data.Items[0].Enable || data.Items[1].Enable {
		t.Errorf("only item with id 3 should toggle, got %+v", data.Items)
	}
}

func TestApplyCallback_InvalidID_NoPanic(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 1, Name: "Milk"}}}
	applyCallback(data, "btn_item_abc")
	if data.Items[0].Enable {
		t.Errorf("invalid id should be a no-op, got %+v", data.Items)
	}
}

func TestApplyCallback_EmptyList(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 1, Name: "Milk"}, {ID: 2, Name: "Bread"}}}
	applyCallback(data, "btn_empty_list")
	if len(data.Items) != 0 {
		t.Errorf("expected empty items, got %+v", data.Items)
	}
}

func TestApplyCallback_RefreshList(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 1, Name: "Milk", Enable: true}, {ID: 2, Name: "Bread", Enable: false}}}
	applyCallback(data, "btn_refresh_list")
	if len(data.Items) != 1 || data.Items[0].Name != "Bread" {
		t.Errorf("expected only disabled item kept, got %+v", data.Items)
	}
}

func TestApplyCallback_Lock(t *testing.T) {
	data := &ChatListData{}
	applyCallback(data, "btn_list_locked")
	if !data.Locked {
		t.Errorf("expected locked, got %+v", data)
	}
	applyCallback(data, "btn_list_locked")
	if data.Locked {
		t.Errorf("expected unlocked, got %+v", data)
	}
}

func TestEnsureItemIDs_LegacyData(t *testing.T) {
	data := &ChatListData{Items: []Item{{Name: "Milk"}, {Name: "Bread"}}}
	ensureItemIDs(data)
	if data.Items[0].ID != 1 || data.Items[1].ID != 2 {
		t.Errorf("expected backfilled IDs 1,2 got %d,%d", data.Items[0].ID, data.Items[1].ID)
	}
	if data.NextItemID != 2 {
		t.Errorf("expected NextItemID 2, got %d", data.NextItemID)
	}
}

func TestEnsureItemIDs_PreservesExisting(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 5, Name: "Milk"}}, NextItemID: 5}
	ensureItemIDs(data)
	if data.Items[0].ID != 5 || data.NextItemID != 5 {
		t.Errorf("existing IDs should be untouched, got %d/%d", data.Items[0].ID, data.NextItemID)
	}
}

func TestEnsureItemIDs_MixedLegacyAndExisting(t *testing.T) {
	data := &ChatListData{Items: []Item{{ID: 7, Name: "Milk"}, {Name: "Bread"}}, NextItemID: 7}
	ensureItemIDs(data)
	if data.Items[0].ID != 7 || data.Items[1].ID != 8 {
		t.Errorf("expected IDs 7,8 got %d,%d", data.Items[0].ID, data.Items[1].ID)
	}
	if data.NextItemID != 8 {
		t.Errorf("expected NextItemID 8, got %d", data.NextItemID)
	}
}

func TestLockChat_SameKeySerializes(t *testing.T) {
	const goroutines = 50
	const increments = 10
	var counter int
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				unlock := lockChat("key")
				counter++
				unlock()
			}
		}()
	}
	wg.Wait()
	want := goroutines * increments
	if counter != want {
		t.Errorf("expected counter %d, got %d (lost updates)", want, counter)
	}
}

func TestLockChat_DifferentKeysIndependent(t *testing.T) {
	unlockA := lockChat("key_a")

	acquired := make(chan struct{})
	go func() {
		unlock := lockChat("key_b")
		close(acquired)
		unlock()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("lock on key_b blocked by key_a: different keys must not contend")
	}

	unlockA()
}
