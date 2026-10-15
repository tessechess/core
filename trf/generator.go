package trf

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ValkoM28/tournamentessentials/core/player"
	"github.com/ValkoM28/tournamentessentials/core/result"
	"github.com/ValkoM28/tournamentessentials/core/round"
)

// errMissingStartingRank is returned when a player has no starting rank assigned.
var errMissingStartingRank = fmt.Errorf("all players must have a StartingRank assigned")

// roundResult holds one player's perspective of a single round.
type roundResult struct {
	opponentRank int  // 0 for bye
	color        byte // 'w', 'b', '-' for bye
	resultCode   byte // TRF result code
}

// Generate builds a TRF 1.6 string and the player mapping from the input data.
// Players must have StartingRank pre-assigned; they are emitted in rank order.
func Generate[ID comparable](input *GeneratorInput[ID]) (string, *PlayerMapping[ID], error) {
	if len(input.Players) == 0 {
		return "", nil, fmt.Errorf("no players provided")
	}
	if input.TotalRounds == 0 {
		return "", nil, fmt.Errorf("total rounds not set")
	}

	// Validate that every player has a starting rank and sort by it.
	ranked := make([]PlayerData[ID], len(input.Players))
	copy(ranked, input.Players)
	for _, p := range ranked {
		if p.StartingRank <= 0 {
			return "", nil, errMissingStartingRank
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].StartingRank < ranked[j].StartingRank
	})

	// Build bidirectional mapping: starting rank <-> player ID.
	mapping := &PlayerMapping[ID]{
		RankToID: make(map[int]ID, len(ranked)),
		IDToRank: make(map[ID]int, len(ranked)),
	}
	for _, p := range ranked {
		mapping.RankToID[p.StartingRank] = p.ID
		mapping.IDToRank[p.ID] = p.StartingRank
	}

	// Build per-player round results from completed rounds.
	playerRounds := make(map[ID]map[int]roundResult, len(ranked))
	for _, g := range input.Games {
		// Skip voided pairings.
		if g.Result != nil && *g.Result == result.Void {
			continue
		}

		ensureMap(playerRounds, g.White)

		if g.Black == nil {
			// Bye — white is the bye recipient.
			playerRounds[g.White][g.Round] = roundResult{
				opponentRank: 0,
				color:        '-',
				resultCode:   resultToTRFBye(g.Result),
			}
			continue
		}

		blackRank, ok := mapping.IDToRank[*g.Black]
		if !ok {
			return "", nil, fmt.Errorf("round %d: black player %v not found in player mapping", g.Round, *g.Black)
		}
		whiteRank, ok := mapping.IDToRank[g.White]
		if !ok {
			return "", nil, fmt.Errorf("round %d: white player %v not found in player mapping", g.Round, g.White)
		}
		ensureMap(playerRounds, *g.Black)

		playerRounds[g.White][g.Round] = roundResult{
			opponentRank: blackRank,
			color:        'w',
			resultCode:   resultToTRFWhite(g.Result),
		}
		playerRounds[*g.Black][g.Round] = roundResult{
			opponentRank: whiteRank,
			color:        'b',
			resultCode:   resultToTRFBlack(g.Result),
		}
	}

	// Apply round settings: inject virtual byes for the upcoming round.
	if input.Settings != nil {
		nextRound := input.CompletedRounds + 1

		// Withdrawn players get a zero-point bye (Z) in the upcoming round.
		for _, id := range input.Settings.WithdrawnPlayers {
			ensureMap(playerRounds, id)
			playerRounds[id][nextRound] = roundResult{
				opponentRank: 0,
				color:        '-',
				resultCode:   'Z',
			}
		}

		// Half-point byes in the upcoming round.
		for _, id := range input.Settings.HalfPointByes {
			ensureMap(playerRounds, id)
			playerRounds[id][nextRound] = roundResult{
				opponentRank: 0,
				color:        '-',
				resultCode:   'H',
			}
		}

		// Late entries: fill missed completed rounds with zero-point byes (Z).
		for _, id := range input.Settings.LateEntries {
			ensureMap(playerRounds, id)
			for rn := 1; rn <= input.CompletedRounds; rn++ {
				if _, exists := playerRounds[id][rn]; !exists {
					playerRounds[id][rn] = roundResult{
						opponentRank: 0,
						color:        '-',
						resultCode:   'Z',
					}
				}
			}
		}
	}

	// Calculate cumulative points per player.
	playerPoints := make(map[ID]float64, len(ranked))
	for id, rounds := range playerRounds {
		var pts float64
		for _, rr := range rounds {
			pts += trfResultPoints(rr.resultCode)
		}
		playerPoints[id] = pts
	}

	// Count rated players.
	ratedCount := 0
	for _, p := range ranked {
		if p.FIDERating > 0 {
			ratedCount++
		}
	}

	// Build the TRF string.
	var sb strings.Builder

	// Tournament header lines.
	sb.WriteString("012 " + input.TournamentName + "\n")
	if input.City != "" {
		sb.WriteString("022 " + input.City + "\n")
	}
	if input.Federation != "" {
		sb.WriteString("032 " + input.Federation + "\n")
	}
	if !input.StartDate.IsZero() {
		sb.WriteString("042 " + formatDate(input.StartDate) + "\n")
	}
	if !input.EndDate.IsZero() {
		sb.WriteString("052 " + formatDate(input.EndDate) + "\n")
	}
	sb.WriteString("062 " + strconv.Itoa(len(ranked)) + "\n")
	sb.WriteString("072 " + strconv.Itoa(ratedCount) + "\n")
	if input.TimeControl != "" {
		sb.WriteString("122 " + input.TimeControl + "\n")
	}
	// XXR extension — required by pairing engines to know total rounds.
	sb.WriteString("XXR " + strconv.Itoa(input.TotalRounds) + "\n")

	// Extension lines from round settings.
	if input.Settings != nil {
		// XXC — first player color preference (round 1 only).
		if input.Settings.FirstPlayerColor == round.ColorWhite {
			sb.WriteString("XXC white1\n")
		} else if input.Settings.FirstPlayerColor == round.ColorBlack {
			sb.WriteString("XXC black1\n")
		}

		// XXF — forbidden pairings (one line per pair with starting rank numbers).
		for _, fp := range input.Settings.ForbiddenPairings {
			rank1, ok1 := mapping.IDToRank[fp.Player1]
			rank2, ok2 := mapping.IDToRank[fp.Player2]
			if ok1 && ok2 {
				sb.WriteString("XXF " + strconv.Itoa(rank1) + " " + strconv.Itoa(rank2) + "\n")
			}
		}
	}

	// Player lines (001), emitted in starting rank order.
	// If settings inject byes for the upcoming round (withdrawn / half-point-bye),
	// extend columns so the engine sees those pre-assigned results.
	roundColumns := input.CompletedRounds
	if input.Settings != nil && (len(input.Settings.WithdrawnPlayers) > 0 || len(input.Settings.HalfPointByes) > 0) {
		roundColumns = input.CompletedRounds + 1
	}
	for _, p := range ranked {
		line := buildPlayerLine(p.StartingRank, &p, playerPoints[p.ID], playerRounds[p.ID], roundColumns)
		sb.WriteString(line + "\n")
	}

	return sb.String(), mapping, nil
}

// sexColumn returns the TRF 1.6 sex character ('m' or 'w'), or 0 when the sex is
// unknown or not recognised.
func sexColumn(s player.Sex) byte {
	switch s {
	case player.SexMale:
		return 'm'
	case player.SexFemale:
		return 'w'
	}
	return 0
}

// buildPlayerLine formats a single 001 line per TRF 1.6 fixed-width positions.
//
// Positions (1-indexed):
//
//	1-3:   "001"
//	5-8:   Starting rank (right-justified 4)
//	10:    Sex
//	11-13: Title (left-justified 3)
//	15-47: Name (left-justified 33)
//	49-52: FIDE rating (right-justified 4)
//	54-56: Federation (left-justified 3)
//	58-68: FIDE number (right-justified 11)
//	70-79: Birth date (left-justified 10)
//	81-84: Points (right-justified 4, e.g. " 3.5")
//	86-89: Rank (left blank for pairing)
//
// Per round (stride 10, starting at position 92):
//
//	+0..+3: Opponent rank (right-justified 4, "0000" for bye)
//	+5:     Color ('w'/'b'/'-')
//	+7:     Result code
func buildPlayerLine[ID comparable](rank int, p *PlayerData[ID], points float64, rounds map[int]roundResult, totalRounds int) string {
	// Buffer size: 89 base positions + 2 gap + totalRounds*10.
	bufLen := 89
	if totalRounds > 0 {
		// Last result code is at position 99+(totalRounds-1)*10, i.e. index 98+(totalRounds-1)*10.
		bufLen = 99 + (totalRounds-1)*10
	}
	buf := make([]byte, bufLen)
	for i := range buf {
		buf[i] = ' '
	}

	// Data identification.
	copy(buf[0:3], "001")

	// Starting rank.
	copy(buf[4:8], rjust(strconv.Itoa(rank), 4))

	// Sex.
	if c := sexColumn(p.Sex); c != 0 {
		buf[9] = c
	}

	// Title.
	if p.Title.FIDE() {
		copy(buf[10:13], ljust(string(p.Title), 3))
	}

	// Name.
	copy(buf[14:47], ljust(p.Name, 33))

	// FIDE rating.
	ratingStr := "0"
	if p.FIDERating > 0 {
		ratingStr = strconv.Itoa(p.FIDERating)
	}
	copy(buf[48:52], rjust(ratingStr, 4))

	// Federation.
	if p.Federation != "" {
		copy(buf[53:56], ljust(p.Federation, 3))
	}

	// FIDE number.
	if p.FIDENumber != "" {
		copy(buf[57:68], rjust(p.FIDENumber, 11))
	}

	// Birth date.
	if p.BirthDate != nil {
		copy(buf[69:79], ljust(formatDate(*p.BirthDate), 10))
	}

	// Points.
	copy(buf[80:84], rjust(formatPoints(points), 4))

	// Rank: left blank (not required for pairing engines).

	// Per-round data.
	for roundNum := 1; roundNum <= totalRounds; roundNum++ {
		offset := 91 + (roundNum-1)*10 // 0-indexed start of this round's block
		rr, ok := rounds[roundNum]
		if !ok {
			// Not paired this round — leave as spaces.
			continue
		}
		// Opponent rank (4 chars).
		if rr.opponentRank == 0 {
			copy(buf[offset:offset+4], "0000")
		} else {
			copy(buf[offset:offset+4], rjust(strconv.Itoa(rr.opponentRank), 4))
		}
		// Color.
		buf[offset+5] = rr.color
		// Result.
		buf[offset+7] = rr.resultCode
	}

	return strings.TrimRight(string(buf), " ")
}

// --- Result code mapping ---

// resultToTRFWhite maps an internal result to a TRF code from white's perspective.
func resultToTRFWhite(r *result.Code) byte {
	if r == nil {
		return ' '
	}
	switch *r {
	case result.WhiteWins:
		return '1'
	case result.BlackWins:
		return '0'
	case result.Draw:
		return '='
	case result.WhiteForfeit:
		return '+'
	case result.BlackForfeit:
		return '-'
	case result.WhiteHalf:
		return '=' // white gets 0.5
	case result.BlackHalf:
		return '0' // white gets 0
	case result.Unplayed:
		return ' '
	default:
		return ' '
	}
}

// resultToTRFBlack maps an internal result to a TRF code from black's perspective.
func resultToTRFBlack(r *result.Code) byte {
	if r == nil {
		return ' '
	}
	switch *r {
	case result.WhiteWins:
		return '0'
	case result.BlackWins:
		return '1'
	case result.Draw:
		return '='
	case result.WhiteForfeit:
		return '-'
	case result.BlackForfeit:
		return '+'
	case result.WhiteHalf:
		return '0' // black gets 0
	case result.BlackHalf:
		return '=' // black gets 0.5
	case result.Unplayed:
		return ' '
	default:
		return ' '
	}
}

// resultToTRFBye maps an internal result to a TRF code for a bye pairing.
func resultToTRFBye(r *result.Code) byte {
	if r == nil {
		return ' '
	}
	switch *r {
	case result.WhiteWins:
		return 'F' // full-point bye
	case result.WhiteHalf:
		return 'H' // half-point bye
	case result.ZeroPointBye:
		return 'Z' // zero-point bye
	case result.WhiteForfeit:
		return '+' // forfeit win (treated as bye with full point)
	case result.Unplayed:
		return ' '
	default:
		return 'Z'
	}
}

// trfResultPoints returns the point value of a TRF result code.
func trfResultPoints(code byte) float64 {
	switch code {
	case '1', '+', 'F', 'U', 'W':
		return 1.0
	case '=', 'H', 'D':
		return 0.5
	default:
		return 0.0
	}
}

// --- Formatting helpers ---

// rjust right-justifies s in a field of width w, truncating from the left if too long.
func rjust(s string, w int) string {
	if len(s) >= w {
		return s[len(s)-w:]
	}
	return strings.Repeat(" ", w-len(s)) + s
}

// ljust left-justifies s in a field of width w, truncating from the right if too long.
func ljust(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

// formatPoints formats points as a TRF points string (e.g. "3.5", "10.0").
func formatPoints(pts float64) string {
	return fmt.Sprintf("%.1f", pts)
}

// ensureMap initialises the inner map for a player if absent.
func ensureMap[ID comparable](m map[ID]map[int]roundResult, id ID) {
	if m[id] == nil {
		m[id] = make(map[int]roundResult)
	}
}

// formatDate formats t as the TRF date YYYY/MM/DD.
func formatDate(t time.Time) string { return t.Format("2006/01/02") }
