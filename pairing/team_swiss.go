package pairing

import (
	"context"
	"fmt"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/result"
	"github.com/tessechess/core/round"
	"github.com/tessechess/core/trf"
)

// TeamInput is everything needed to pair the next Swiss round of a team
// section.
type TeamInput[TeamID, PlayerID comparable] struct {
	Tournament      Tournament                      // header data written to the TRF
	BoardsPerMatch  int                             // boards per match; must be >= 1
	Teams           []Team[TeamID, PlayerID]        // confirmed teams of the section
	History         []round.Match[TeamID, PlayerID] // matches of the completed rounds, in round order
	CompletedRounds int                             // number of completed rounds; the round being paired is CompletedRounds+1
	Settings        *Settings[TeamID]               // arbiter settings at the team level; nil = none (round 1 requires FirstPlayerColor)
}

// PreparedTeams is a team round ready to be sent to a pairing engine.
type PreparedTeams[TeamID, PlayerID comparable] struct {
	trf         string
	mapping     *trf.PlayerMapping[TeamID]
	completed   int
	fixedTables []FixedTable[TeamID]
}

// TRF returns the TRF input that Run sends to the engine.
func (p *PreparedTeams[TeamID, PlayerID]) TRF() string { return p.trf }

// PairTeams pairs the next Swiss round of a team tournament with the given
// engine: it calls PrepareTeams and then Run, and returns their errors.
//
// The result has one match per table, ordered by Table number. Round is
// in.CompletedRounds+1, Table is set, Away is nil for a bye, WhiteOnFirstBoard
// is true on played matches (so Home has white on board 1), Boards is nil
// (use PairMatchBoards once team lineups are known), and Result is nil. Fixed
// tables are applied.
//
// Errors can be tested with errors.Is: ErrInvalidBoardsPerMatch,
// ErrInvalidSettings, ErrTRFGeneration, the engine errors of package engine,
// ErrTRFParsing and ErrUnknownEngineRank.
func PairTeams[TeamID, PlayerID comparable](
	ctx context.Context,
	eng engine.PairingEngine,
	in TeamInput[TeamID, PlayerID],
) ([]round.Match[TeamID, PlayerID], error) {
	p, err := PrepareTeams(in)
	if err != nil {
		return nil, err
	}
	return p.Run(ctx, eng)
}

// PrepareTeams merges team-level fixed tables and round-level fixed boards into
// the table assignments, converts teams and completed matches into TRF data,
// and generates the TRF input without running an engine.
//
// Each team's TRF rating is computed with TeamRating(team.Players,
// in.BoardsPerMatch). Each completed match in in.History is converted into a
// team-level round.Game where the team with white on board 1 is White and the
// match outcome (win, draw, loss, forfeit, or bye) becomes the game's result.
func PrepareTeams[TeamID, PlayerID comparable](in TeamInput[TeamID, PlayerID]) (*PreparedTeams[TeamID, PlayerID], error) {
	if in.BoardsPerMatch < 1 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidBoardsPerMatch, in.BoardsPerMatch)
	}

	settings := Settings[TeamID]{}
	if in.Settings != nil {
		settings = *in.Settings
	}

	if in.CompletedRounds == 0 && settings.FirstPlayerColor != round.ColorWhite && settings.FirstPlayerColor != round.ColorBlack {
		return nil, fmt.Errorf("%w: first_player_color must be 'white' or 'black' for round 1", ErrInvalidSettings)
	}

	// Merge round-level FixedBoards and team-level FixedTable into fixedTables.
	// Round-level entries take priority; team-level entries fill in the rest.
	fixedTables := make([]FixedTable[TeamID], 0, len(settings.FixedBoards))
	for _, fb := range settings.FixedBoards {
		fixedTables = append(fixedTables, FixedTable[TeamID]{
			TableNumber: fb.BoardNumber,
			Team:        fb.Player,
		})
	}
	for _, tm := range in.Teams {
		if tm.FixedTable == nil {
			continue
		}
		alreadyFixed := false
		for _, ft := range fixedTables {
			if ft.Team == tm.ID {
				alreadyFixed = true
				break
			}
		}
		if !alreadyFixed {
			fixedTables = append(fixedTables, FixedTable[TeamID]{
				TableNumber: *tm.FixedTable,
				Team:        tm.ID,
			})
		}
	}

	players := make([]trf.PlayerData[TeamID], 0, len(in.Teams))
	for _, tm := range in.Teams {
		players = append(players, trf.PlayerData[TeamID]{
			ID:           tm.ID,
			StartingRank: tm.StartingRank,
			Name:         tm.Name,
			FIDERating:   TeamRating(tm.Players, in.BoardsPerMatch),
			Federation:   tm.Federation,
		})
	}

	games := make([]round.Game[TeamID], 0, len(in.History))
	for _, m := range in.History {
		games = append(games, matchToTeamGame(m))
	}

	input := &trf.GeneratorInput[TeamID]{
		TournamentName:  in.Tournament.Name,
		City:            in.Tournament.City,
		Federation:      in.Tournament.Federation,
		StartDate:       in.Tournament.StartDate,
		EndDate:         in.Tournament.EndDate,
		TimeControl:     in.Tournament.TimeControl,
		TotalRounds:     in.Tournament.TotalRounds,
		Players:         players,
		Games:           games,
		CompletedRounds: in.CompletedRounds,
		Settings:        toTRFSettings(settings),
	}

	trfString, mapping, err := trf.Generate(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTRFGeneration, err)
	}

	return &PreparedTeams[TeamID, PlayerID]{
		trf:         trfString,
		mapping:     mapping,
		completed:   in.CompletedRounds,
		fixedTables: fixedTables,
	}, nil
}

// Run sends the TRF to the engine, parses its answer with the parser matching
// eng.OutputFormat(), maps ranks back to team IDs and applies fixed tables.
// The returned matches are ordered by Table, with Round set to the round being
// paired, WhiteOnFirstBoard set to true on non-bye matches, and Result nil.
func (p *PreparedTeams[TeamID, PlayerID]) Run(
	ctx context.Context,
	eng engine.PairingEngine,
) ([]round.Match[TeamID, PlayerID], error) {
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

	matches := make([]round.Match[TeamID, PlayerID], 0, len(parsed))
	for _, pp := range parsed {
		home, ok := p.mapping.RankToID[pp.WhiteRank]
		if !ok {
			return nil, fmt.Errorf("%w: white rank %d", ErrUnknownEngineRank, pp.WhiteRank)
		}
		var away *TeamID
		if pp.BlackRank > 0 {
			id, ok := p.mapping.RankToID[pp.BlackRank]
			if !ok {
				return nil, fmt.Errorf("%w: black rank %d", ErrUnknownEngineRank, pp.BlackRank)
			}
			away = &id
		}
		matches = append(matches, round.Match[TeamID, PlayerID]{
			Round:             p.completed + 1,
			Table:             pp.BoardNumber,
			Home:              home,
			Away:              away,
			WhiteOnFirstBoard: away != nil,
		})
	}

	if len(p.fixedTables) > 0 {
		matches = ApplyFixedTables(matches, p.fixedTables)
	}
	return matches, nil
}

// matchToTeamGame converts a completed team Match into a team-level round.Game
// for TRF generation. The team with white on board 1 becomes Game.White.
func matchToTeamGame[TeamID, PlayerID comparable](m round.Match[TeamID, PlayerID]) round.Game[TeamID] {
	if m.IsBye() {
		return round.Game[TeamID]{
			Round:  m.Round,
			Board:  m.Table,
			White:  m.Home,
			Black:  nil,
			Result: m.Result,
		}
	}

	whiteTeam := m.Home
	blackTeam := *m.Away
	if !m.WhiteOnFirstBoard {
		whiteTeam = *m.Away
		blackTeam = m.Home
	}

	var res *result.Code
	if m.Result != nil {
		c := *m.Result
		if !m.WhiteOnFirstBoard {
			c = flipResultCode(c)
		}
		res = &c
	} else if m.Settled() {
		homePts, awayPts := m.GamePoints()
		whitePts, blackPts := homePts, awayPts
		if !m.WhiteOnFirstBoard {
			whitePts, blackPts = awayPts, homePts
		}
		var c result.Code
		switch {
		case whitePts > blackPts:
			c = result.WhiteWins
		case whitePts < blackPts:
			c = result.BlackWins
		default:
			c = result.Draw
		}
		res = &c
	}

	return round.Game[TeamID]{
		Round:  m.Round,
		Board:  m.Table,
		White:  whiteTeam,
		Black:  &blackTeam,
		Result: res,
	}
}

// flipResultCode reverses a result code between white (Home) and black (Away).
func flipResultCode(c result.Code) result.Code {
	switch c {
	case result.WhiteWins:
		return result.BlackWins
	case result.BlackWins:
		return result.WhiteWins
	case result.WhiteForfeit:
		return result.BlackForfeit
	case result.BlackForfeit:
		return result.WhiteForfeit
	case result.WhiteHalf:
		return result.BlackHalf
	case result.BlackHalf:
		return result.WhiteHalf
	default:
		return c
	}
}
