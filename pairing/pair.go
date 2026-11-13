package pairing

import (
	"context"
	"fmt"
	"sort"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/round"
	"github.com/tessechess/core/trf"
)

// Pair pairs the next round with the given engine: it calls Prepare and then
// Run, and returns their errors.
//
// The result has one game per board, ordered by board number. Round is
// in.CompletedRounds+1, Board is set, Black is nil for a bye, and Result is nil
// (nothing has been played). Players with a bye already assigned for this round
// (withdrawn, half-point bye) are left to the engine's input and are not paired
// by it. Fixed boards are applied.
//
// Errors can be tested with errors.Is: ErrInvalidSettings (round 1 is being
// paired without a valid Settings.FirstPlayerColor), ErrTRFGeneration (the
// input could not be turned into a TRF, e.g. a player has no StartingRank,
// TotalRounds is 0 or a game names an unknown player), the engine errors of
// package engine (engine not found, timeout, non-zero exit), ErrTRFParsing and
// ErrUnknownEngineRank (the engine's answer could not be read).
func Pair[ID comparable](ctx context.Context, eng engine.PairingEngine, in Input[ID]) ([]round.Game[ID], error) {
	p, err := Prepare(in)
	if err != nil {
		return nil, err
	}
	return p.Run(ctx, eng)
}

// Prepared is a round ready to be sent to a pairing engine.
type Prepared[ID comparable] struct {
	trf         string
	mapping     *trf.PlayerMapping[ID]
	completed   int
	fixedBoards []FixedBoard[ID]
}

// TRF returns the TRF input that Run sends to the engine.
func (p *Prepared[ID]) TRF() string { return p.trf }

// Prepare merges player-level fixed boards into the settings and generates the
// TRF input, without running an engine. Use it to preview the TRF (see
// Prepared.TRF) or to run the engine separately. When pairing round 1
// (CompletedRounds == 0), Settings.FirstPlayerColor must be round.ColorWhite or
// round.ColorBlack; otherwise Prepare returns ErrInvalidSettings. Generation
// errors are wrapped with ErrTRFGeneration. The caller's settings are not
// modified.
func Prepare[ID comparable](in Input[ID]) (*Prepared[ID], error) {
	settings := Settings[ID]{}
	if in.Settings != nil {
		settings = *in.Settings
	}

	if in.CompletedRounds == 0 && settings.FirstPlayerColor != round.ColorWhite && settings.FirstPlayerColor != round.ColorBlack {
		return nil, fmt.Errorf("%w: first_player_color must be 'white' or 'black' for round 1", ErrInvalidSettings)
	}

	// Merge player-level fixed boards into the round settings. Round-level
	// entries take priority; player-level entries fill in the rest.
	fixedBoards := append([]FixedBoard[ID](nil), settings.FixedBoards...)
	for _, pl := range in.Players {
		if pl.FixedBoard == nil {
			continue
		}
		alreadyFixed := false
		for _, fb := range fixedBoards {
			if fb.Player == pl.ID {
				alreadyFixed = true
				break
			}
		}
		if !alreadyFixed {
			fixedBoards = append(fixedBoards, FixedBoard[ID]{BoardNumber: *pl.FixedBoard, Player: pl.ID})
		}
	}

	players := make([]trf.PlayerData[ID], 0, len(in.Players))
	for _, pl := range in.Players {
		players = append(players, trf.PlayerData[ID]{
			ID:           pl.ID,
			StartingRank: pl.StartingRank,
			Name:         pl.LastName + ", " + pl.FirstName,
			Title:        pl.Title,
			FIDERating:   pl.Rating,
			Federation:   pl.Federation,
			FIDENumber:   pl.FIDENumber,
			BirthDate:    pl.Birthday,
			Sex:          pl.Sex,
		})
	}

	input := &trf.GeneratorInput[ID]{
		TournamentName:  in.Tournament.Name,
		City:            in.Tournament.City,
		Federation:      in.Tournament.Federation,
		StartDate:       in.Tournament.StartDate,
		EndDate:         in.Tournament.EndDate,
		TimeControl:     in.Tournament.TimeControl,
		TotalRounds:     in.Tournament.TotalRounds,
		Players:         players,
		Games:           in.History,
		CompletedRounds: in.CompletedRounds,
		Settings:        toTRFSettings(settings),
	}

	trfString, mapping, err := trf.Generate(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTRFGeneration, err)
	}

	return &Prepared[ID]{
		trf:         trfString,
		mapping:     mapping,
		completed:   in.CompletedRounds,
		fixedBoards: fixedBoards,
	}, nil
}

// Run sends the TRF to the engine, parses its answer with the parser matching
// eng.OutputFormat(), maps ranks back to player IDs and applies fixed boards.
// The returned games are ordered by board, with Round set to the round being
// paired and Result nil (see Pair).
//
// Errors from the engine are returned as they are (see package engine). A
// parse failure, including an unknown OutputFormat, is wrapped with
// ErrTRFParsing, and a rank in the answer that is not in the TRF with
// ErrUnknownEngineRank.
func (p *Prepared[ID]) Run(ctx context.Context, eng engine.PairingEngine) ([]round.Game[ID], error) {
	output, err := eng.GeneratePairings(ctx, p.trf)
	if err != nil {
		return nil, err
	}

	var parsed []trf.ParsedPairing
	switch eng.OutputFormat() {
	case engine.OutputTRF:
		parsed, err = trf.ParseTRFRound(output, p.completed)
	case engine.OutputSimplePairs:
		parsed, err = trf.ParseSimplePairs(output)
	default:
		return nil, fmt.Errorf("%w: unknown engine output format %q", ErrTRFParsing, eng.OutputFormat())
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTRFParsing, err)
	}

	boards := make([]round.Game[ID], 0, len(parsed))
	for _, pp := range parsed {
		white, ok := p.mapping.RankToID[pp.WhiteRank]
		if !ok {
			return nil, fmt.Errorf("%w: white rank %d", ErrUnknownEngineRank, pp.WhiteRank)
		}
		var black *ID
		if pp.BlackRank > 0 {
			id, ok := p.mapping.RankToID[pp.BlackRank]
			if !ok {
				return nil, fmt.Errorf("%w: black rank %d", ErrUnknownEngineRank, pp.BlackRank)
			}
			black = &id
		}
		boards = append(boards, round.Game[ID]{Round: p.completed + 1, Board: pp.BoardNumber, White: white, Black: black})
	}

	if len(p.fixedBoards) > 0 {
		boards = ApplyFixedBoards(boards, p.fixedBoards)
	}
	return boards, nil
}

// toTRFSettings translates round settings to TRF terms. Fixed boards are not
// part of the TRF; they are applied after the engine has paired the round.
func toTRFSettings[ID comparable](s Settings[ID]) *trf.RoundSettings[ID] {
	rs := &trf.RoundSettings[ID]{
		WithdrawnPlayers: s.WithdrawnPlayers,
		HalfPointByes:    s.HalfPointByes,
		FirstPlayerColor: s.FirstPlayerColor,
		LateEntries:      s.LateEntries,
		// TODO(#377): emit acceleration (XXA lines) from s.Acceleration.
	}
	for _, fp := range s.ForbiddenPairings {
		rs.ForbiddenPairings = append(rs.ForbiddenPairings, trf.ForbiddenPair[ID]{
			Player1: fp.Player1,
			Player2: fp.Player2,
		})
	}
	return rs
}

// ApplyFixedBoards moves designated players to their fixed board numbers.
// For each FixedBoard, the player's game is moved to the target board number
// and all other board numbers are re-assigned sequentially around them.
func ApplyFixedBoards[ID comparable](boards []round.Game[ID], fixedBoards []FixedBoard[ID]) []round.Game[ID] {
	// Build a lookup: player -> target board.
	playerToBoard := make(map[ID]int, len(fixedBoards))
	for _, fb := range fixedBoards {
		playerToBoard[fb.Player] = fb.BoardNumber
	}

	// Separate fixed games from the rest.
	var fixed []round.Game[ID]
	var rest []round.Game[ID]
	for _, b := range boards {
		if _, ok := playerToBoard[b.White]; ok {
			fixed = append(fixed, b)
			continue
		}
		if b.Black != nil {
			if _, ok := playerToBoard[*b.Black]; ok {
				fixed = append(fixed, b)
				continue
			}
		}
		rest = append(rest, b)
	}

	// Assign target board numbers to fixed games.
	for i := range fixed {
		if board, ok := playerToBoard[fixed[i].White]; ok {
			fixed[i].Board = board
		} else if fixed[i].Black != nil {
			if board, ok := playerToBoard[*fixed[i].Black]; ok {
				fixed[i].Board = board
			}
		}
	}

	// Collect taken board numbers.
	taken := make(map[int]bool, len(fixed))
	for _, b := range fixed {
		taken[b.Board] = true
	}

	// Re-number remaining games, filling gaps.
	board := 1
	for i := range rest {
		for taken[board] {
			board++
		}
		rest[i].Board = board
		board++
	}

	// Merge and sort by board number.
	all := append(fixed, rest...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Board < all[j].Board
	})
	return all
}
