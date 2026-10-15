package pairing

import (
	"context"
	"fmt"
	"sort"

	"github.com/ValkoM28/tournamentessentials/core/engine"
	"github.com/ValkoM28/tournamentessentials/core/trf"
)

// Pair pairs the next round: Prepare followed by Run.
func Pair[ID comparable](ctx context.Context, eng engine.PairingEngine, in Input[ID]) ([]Board[ID], error) {
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
	historyLen  int
	fixedBoards []FixedBoard[ID]
}

// TRF returns the TRF input that Run sends to the engine.
func (p *Prepared[ID]) TRF() string { return p.trf }

// Prepare merges player-level fixed boards into the settings and generates the
// TRF input. Generation errors are wrapped with ErrTRFGeneration.
func Prepare[ID comparable](in Input[ID]) (*Prepared[ID], error) {
	settings := Settings[ID]{}
	if in.Settings != nil {
		settings = *in.Settings
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
		bd := ""
		if pl.Birthday != nil {
			bd = pl.Birthday.Format("2006/01/02")
		}
		players = append(players, trf.PlayerData[ID]{
			ID:           pl.ID,
			StartingRank: pl.StartingRank,
			Name:         pl.LastName + ", " + pl.FirstName,
			Title:        pl.Title,
			FIDERating:   pl.Rating,
			Federation:   pl.Federation,
			FIDENumber:   pl.FIDENumber,
			BirthDate:    bd,
			Sex:          pl.Sex,
		})
	}

	input := &trf.GeneratorInput[ID]{
		TournamentName: in.Tournament.Name,
		City:           in.Tournament.City,
		Federation:     in.Tournament.Federation,
		StartDate:      in.Tournament.StartDate.Format("2006/01/02"),
		EndDate:        in.Tournament.EndDate.Format("2006/01/02"),
		TimeControl:    in.Tournament.TimeControl,
		TotalRounds:    in.Tournament.TotalRounds,
		Players:        players,
		Rounds:         in.History,
		Settings:       toTRFSettings(settings),
	}

	trfString, mapping, err := trf.Generate(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTRFGeneration, err)
	}

	return &Prepared[ID]{
		trf:         trfString,
		mapping:     mapping,
		historyLen:  len(in.History),
		fixedBoards: fixedBoards,
	}, nil
}

// Run sends the TRF to the engine, parses its answer with the parser matching
// eng.OutputFormat(), maps ranks back to player IDs and applies fixed boards.
// Engine errors are returned as they are; parse errors are wrapped with
// ErrTRFParsing and unknown ranks with ErrUnknownEngineRank.
func (p *Prepared[ID]) Run(ctx context.Context, eng engine.PairingEngine) ([]Board[ID], error) {
	output, err := eng.GeneratePairings(ctx, p.trf)
	if err != nil {
		return nil, err
	}

	var parsed []trf.ParsedPairing
	switch eng.OutputFormat() {
	case engine.OutputTRF:
		parsed, err = trf.ParseBBPPairingsOutput(output, p.historyLen)
	case engine.OutputSimplePairs:
		parsed, err = trf.ParseJaVaFoOutput(output)
	default:
		return nil, fmt.Errorf("%w: unknown engine output format %q", ErrTRFParsing, eng.OutputFormat())
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTRFParsing, err)
	}

	boards := make([]Board[ID], 0, len(parsed))
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
		boards = append(boards, Board[ID]{Number: pp.BoardNumber, White: white, Black: black})
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
	}
	for _, fp := range s.ForbiddenPairings {
		rs.ForbiddenPairings = append(rs.ForbiddenPairings, trf.ForbiddenPairTRF[ID]{
			Player1: fp.Player1,
			Player2: fp.Player2,
		})
	}
	return rs
}

// ApplyFixedBoards moves designated players to their fixed board numbers.
// For each FixedBoard, the player's game is moved to the target board number
// and all other board numbers are re-assigned sequentially around them.
func ApplyFixedBoards[ID comparable](boards []Board[ID], fixedBoards []FixedBoard[ID]) []Board[ID] {
	// Build a lookup: player -> target board.
	playerToBoard := make(map[ID]int, len(fixedBoards))
	for _, fb := range fixedBoards {
		playerToBoard[fb.Player] = fb.BoardNumber
	}

	// Separate fixed games from the rest.
	var fixed []Board[ID]
	var rest []Board[ID]
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
			fixed[i].Number = board
		} else if fixed[i].Black != nil {
			if board, ok := playerToBoard[*fixed[i].Black]; ok {
				fixed[i].Number = board
			}
		}
	}

	// Collect taken board numbers.
	taken := make(map[int]bool, len(fixed))
	for _, b := range fixed {
		taken[b.Number] = true
	}

	// Re-number remaining games, filling gaps.
	board := 1
	for i := range rest {
		for taken[board] {
			board++
		}
		rest[i].Number = board
		board++
	}

	// Merge and sort by board number.
	all := append(fixed, rest...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Number < all[j].Number
	})
	return all
}
