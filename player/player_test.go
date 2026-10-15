package player_test

import (
	"testing"

	"github.com/ValkoM28/tournamentessentials/core/player"
)

func TestTitleOrder(t *testing.T) {
	fide := []player.Title{player.GM, player.IM, player.WGM, player.FM, player.WIM, player.CM, player.WFM, player.WCM}
	for i, title := range fide {
		if got := title.Order(); got != i {
			t.Errorf("%q.Order() = %d, want %d", title, got, i)
		}
	}
	untitled := player.NoTitle.Order()
	if untitled <= player.WCM.Order() {
		t.Fatalf("untitled (%d) must rank after WCM", untitled)
	}
	for _, title := range []player.Title{player.AGM, player.AIM, player.AFM, player.ACM, "NM", "gm", "unknown"} {
		if got := title.Order(); got != untitled {
			t.Errorf("%q.Order() = %d, want untitled (%d)", title, got, untitled)
		}
	}
}

func TestTitleFIDE(t *testing.T) {
	for _, title := range []player.Title{player.GM, player.IM, player.WGM, player.FM, player.WIM, player.CM, player.WFM, player.WCM} {
		if !title.FIDE() {
			t.Errorf("%q.FIDE() = false, want true", title)
		}
	}
	for _, title := range []player.Title{player.NoTitle, player.AGM, player.AIM, player.AFM, player.ACM, "gm", "unknown"} {
		if title.FIDE() {
			t.Errorf("%q.FIDE() = true, want false", title)
		}
	}
}
