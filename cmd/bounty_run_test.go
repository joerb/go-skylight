package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func bountyMockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/chores") && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"data":{"id":"c1","attributes":{"summary":"Dishes","reward_points":10}}}`)
		case strings.HasSuffix(r.URL.Path, "/chores") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"data":[{"id":"c1","attributes":{"summary":"Dishes","status":"pending","reward_points":10}}]}`)
		case strings.HasSuffix(r.URL.Path, "/c1") && r.Method == http.MethodPut:
			fmt.Fprint(w, `{"data":{"id":"c1","attributes":{"summary":"Updated","reward_points":10}}}`)
		case strings.HasSuffix(r.URL.Path, "/c1") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/rewards") && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"data":[{"id":"r1","attributes":{"name":"Ice cream","point_value":10}}]}`)
		case strings.HasSuffix(r.URL.Path, "/rewards") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"data":[{"id":"r1","attributes":{"name":"Ice cream","point_value":10}}]}`)
		case strings.HasSuffix(r.URL.Path, "/r1") && r.Method == http.MethodPatch:
			fmt.Fprint(w, `{"data":{"id":"r1","attributes":{"name":"Updated Reward","point_value":10}}}`)
		case strings.HasSuffix(r.URL.Path, "/r1") && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestBountyCreateCmd(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())
	origTitle, origPoints, origRewardTitle := bountyTitle, bountyPoints, bountyRewardTitle
	bountyTitle, bountyPoints, bountyRewardTitle = "Dishes", 10, "Ice cream"
	t.Cleanup(func() { bountyTitle, bountyPoints, bountyRewardTitle = origTitle, origPoints, origRewardTitle })

	out := captureStdout(func() {
		if err := bountyCreateCmd.RunE(bountyCreateCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Dishes") || !strings.Contains(out, "Ice cream") {
		t.Errorf("expected chore and reward in output, got: %s", out)
	}
}

func TestBountyListCmd(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())

	out := captureStdout(func() {
		if err := bountyListCmd.RunE(bountyListCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Dishes") {
		t.Errorf("expected matched bounty in output, got: %s", out)
	}
}

func TestBountyDeleteCmd(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())
	origChoreID, origRewardID, origYes := bountyChoreID, bountyRewardID, yes
	bountyChoreID, bountyRewardID, yes = "c1", "r1", true
	t.Cleanup(func() { bountyChoreID, bountyRewardID, yes = origChoreID, origRewardID, origYes })

	out := captureStdout(func() {
		if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Bounty deleted successfully") {
		t.Errorf("expected deletion confirmation, got: %s", out)
	}
}

func TestBountyDeleteCmd_DryRun(t *testing.T) {
	origChoreID, origRewardID, origDryRun := bountyChoreID, bountyRewardID, dryRun
	bountyChoreID, bountyRewardID, dryRun = "c1", "r1", true
	t.Cleanup(func() { bountyChoreID, bountyRewardID, dryRun = origChoreID, origRewardID, origDryRun })

	origFrameID := frameID
	frameID = "test-frame"
	t.Cleanup(func() { frameID = origFrameID })

	out := captureStdout(func() {
		if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Dry run") {
		t.Errorf("expected dry run output, got: %s", out)
	}
}

func TestBountyUpdateCmd(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())
	origChoreID, origRewardID, origTitle := bountyChoreID, bountyRewardID, bountyTitle
	bountyChoreID, bountyRewardID, bountyTitle = "c1", "r1", "Updated"
	t.Cleanup(func() { bountyChoreID, bountyRewardID, bountyTitle = origChoreID, origRewardID, origTitle })

	// pflag.Set() marks the flag as permanently "changed" on the shared
	// command singleton (no unset API), so this only runs once per process.
	if err := bountyUpdateCmd.Flags().Set("title", "Updated"); err != nil {
		t.Fatalf("setting title flag: %v", err)
	}

	out := captureStdout(func() {
		if err := bountyUpdateCmd.RunE(bountyUpdateCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Updated") {
		t.Errorf("expected updated bounty in output, got: %s", out)
	}
}

func TestBountyCreateCmd_InvalidDate(t *testing.T) {
	newCmdTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot) // won't be reached
	})
	orig := bountyDueDate
	bountyDueDate = "not-a-date"
	t.Cleanup(func() { bountyDueDate = orig })

	err := bountyCreateCmd.RunE(bountyCreateCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
	if !strings.Contains(err.Error(), "due-date") && !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected date validation error, got: %v", err)
	}
}

func TestBountyListCmd_InvalidAfterDate(t *testing.T) {
	origAfter := bountyAfter
	bountyAfter = "not-a-date"
	t.Cleanup(func() { bountyAfter = origAfter })

	origFrameID := frameID
	frameID = "test-frame"
	t.Cleanup(func() { frameID = origFrameID })

	err := bountyListCmd.RunE(bountyListCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid --after date, got nil")
	}
	if !strings.Contains(err.Error(), "--after") {
		t.Errorf("expected --after in error, got: %v", err)
	}
}

func TestBountyListCmd_InvalidBeforeDate(t *testing.T) {
	origBefore := bountyBefore
	bountyBefore = "not-a-date"
	t.Cleanup(func() { bountyBefore = origBefore })

	origFrameID := frameID
	frameID = "test-frame"
	t.Cleanup(func() { frameID = origFrameID })

	err := bountyListCmd.RunE(bountyListCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid --before date, got nil")
	}
	if !strings.Contains(err.Error(), "--before") {
		t.Errorf("expected --before in error, got: %v", err)
	}
}

func TestBountyDeleteAll(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())
	origAll, origYes := bountyAll, yes
	bountyAll, yes = true, true
	t.Cleanup(func() { bountyAll, yes = origAll, origYes })

	out := captureStdout(func() {
		if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Deleted") {
		t.Errorf("expected deletion confirmation, got: %s", out)
	}
}

func TestBountyDeleteAll_DryRun(t *testing.T) {
	newCmdTestClient(t, bountyMockHandler())
	origAll, origDryRun := bountyAll, dryRun
	bountyAll, dryRun = true, true
	t.Cleanup(func() { bountyAll, dryRun = origAll, origDryRun })

	out := captureStdout(func() {
		if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Dry run") {
		t.Errorf("expected dry run output, got: %s", out)
	}
}

func TestBountyDeleteAll_InvalidDate(t *testing.T) {
	tests := []struct {
		name      string
		after     string
		before    string
		wantInErr string
	}{
		{name: "invalid after", after: "not-a-date", wantInErr: "--after"},
		{name: "invalid before", before: "not-a-date", wantInErr: "--before"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newCmdTestClient(t, http500Handler)
			origAll, origAfter, origBefore := bountyAll, bountyAfter, bountyBefore
			bountyAll, bountyAfter, bountyBefore = true, tc.after, tc.before
			t.Cleanup(func() {
				bountyAll, bountyAfter, bountyBefore = origAll, origAfter, origBefore
			})

			err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil)
			if err == nil {
				t.Fatal("expected error for invalid date, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("expected %q in error, got: %v", tc.wantInErr, err)
			}
		})
	}
}

func TestBountyDeleteAll_NoMatchesUsesDateRange(t *testing.T) {
	var gotAfter, gotBefore string
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/chores") && r.Method == http.MethodGet:
			gotAfter = r.URL.Query().Get("after")
			gotBefore = r.URL.Query().Get("before")
			fmt.Fprint(w, `{"data":[]}`)
		case strings.HasSuffix(r.URL.Path, "/rewards") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"data":[]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
	newCmdTestClient(t, handler)
	origAll, origAfter, origBefore, origYes, origDryRun := bountyAll, bountyAfter, bountyBefore, yes, dryRun
	bountyAll, bountyAfter, bountyBefore, yes, dryRun = true, "2026-08-01", "2026-10-31", true, false
	t.Cleanup(func() {
		bountyAll, bountyAfter, bountyBefore, yes, dryRun = origAll, origAfter, origBefore, origYes, origDryRun
	})

	out := captureStdout(func() {
		if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No bounties found to delete") {
		t.Errorf("expected no-matches message, got: %s", out)
	}
	if gotAfter != "2026-08-01" || gotBefore != "2026-10-31" {
		t.Errorf("expected date range 2026-08-01..2026-10-31, got %q..%q", gotAfter, gotBefore)
	}
}

func TestBountyDeleteAll_ListError(t *testing.T) {
	newCmdTestClient(t, http500Handler)
	origAll := bountyAll
	bountyAll = true
	t.Cleanup(func() { bountyAll = origAll })

	err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil)
	if err == nil {
		t.Fatal("expected listing error, got nil")
	}
	if !strings.Contains(err.Error(), "listing bounties") {
		t.Errorf("expected listing error context, got: %v", err)
	}
}

func TestBountyDeleteAll_ConfirmationDeclined(t *testing.T) {
	var deleteCalls int
	mock := bountyMockHandler()
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCalls++
		}
		mock.ServeHTTP(w, r)
	}
	newCmdTestClient(t, handler)
	origAll, origYes := bountyAll, yes
	bountyAll, yes = true, false
	t.Cleanup(func() { bountyAll, yes = origAll, origYes })
	mockStdin(t, "n\n")

	var out string
	captureStderr(func() {
		out = captureStdout(func() {
			if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	})
	if strings.Contains(out, "Deleted") {
		t.Errorf("expected no deletion output after declining confirmation, got: %s", out)
	}
	if deleteCalls != 0 {
		t.Errorf("expected no delete requests after declining confirmation, got %d", deleteCalls)
	}
}

func TestBountyDeleteAll_ContinuesAfterDeleteFailure(t *testing.T) {
	var deletePaths []string
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/chores") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"data":[{"id":"c1","attributes":{"summary":"Dishes","status":"pending","reward_points":10}},{"id":"c2","attributes":{"summary":"Laundry","status":"pending","reward_points":20}}]}`)
		case strings.HasSuffix(r.URL.Path, "/rewards") && r.Method == http.MethodGet:
			fmt.Fprint(w, `{"data":[{"id":"r1","attributes":{"name":"Ice cream","point_value":10}},{"id":"r2","attributes":{"name":"Movie","point_value":20}}]}`)
		case strings.Contains(r.URL.Path, "/chores/") && r.Method == http.MethodDelete:
			deletePaths = append(deletePaths, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/rewards/") && r.Method == http.MethodDelete:
			deletePaths = append(deletePaths, r.URL.Path)
			if strings.HasSuffix(r.URL.Path, "/r1") {
				http.Error(w, "temporary failure", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
	newCmdTestClient(t, handler)
	origAll, origYes := bountyAll, yes
	bountyAll, yes = true, true
	t.Cleanup(func() { bountyAll, yes = origAll, origYes })

	var out, stderr string
	out = captureStdout(func() {
		stderr = captureStderr(func() {
			if err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	})
	if !strings.Contains(out, "Deleted 1/2 bounties") {
		t.Errorf("expected one successful deletion, got: %s", out)
	}
	if !strings.Contains(stderr, "WARNING: failed to delete bounty") {
		t.Errorf("expected warning for failed deletion, got: %s", stderr)
	}
	if len(deletePaths) != 4 || !strings.HasSuffix(deletePaths[2], "/c2") || !strings.HasSuffix(deletePaths[3], "/r2") {
		t.Errorf("expected deletion attempts to continue with second bounty, got: %v", deletePaths)
	}
}

func TestBountyDeleteCmd_MissingIDs(t *testing.T) {
	origChoreID, origRewardID, origAll := bountyChoreID, bountyRewardID, bountyAll
	bountyChoreID, bountyRewardID, bountyAll = "", "", false
	t.Cleanup(func() { bountyChoreID, bountyRewardID, bountyAll = origChoreID, origRewardID, origAll })

	origFrameID := frameID
	frameID = "test-frame"
	t.Cleanup(func() { frameID = origFrameID })

	err := bountyDeleteCmd.RunE(bountyDeleteCmd, nil)
	if err == nil {
		t.Fatal("expected error for missing IDs without --all, got nil")
	}
	if !strings.Contains(err.Error(), "--chore-id") {
		t.Errorf("expected error mentioning --chore-id, got: %v", err)
	}
}

func TestBountyCmdExists(t *testing.T) {
	assertCommandRegistered(t, rootCmd, "bounty")
}
