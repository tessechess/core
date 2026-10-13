package trf

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseBBPPairingsOutput extracts the new round's pairings from a TRF output in
// which a new round column is appended to each player line (engine.OutputTRF).
// Despite the name, bbpPairings' -p output is the simple pair list: parse that
// with ParseJaVaFoOutput. inputRoundCount is the
// number of completed rounds that were in the input TRF; the new pairings are at
// round inputRoundCount+1.
func ParseBBPPairingsOutput(trfOutput string, inputRoundCount int) ([]ParsedPairing, error) {
	newRound := inputRoundCount + 1
	offset := 91 + (newRound-1)*10 // 0-indexed start of the new round's data

	lines := strings.Split(trfOutput, "\n")

	var pairings []ParsedPairing
	boardNum := 0

	for _, line := range lines {
		if len(line) < 4 || line[0:3] != "001" {
			continue
		}

		// Starting rank: positions 5-8 (indices 4-7).
		rank, err := parseIntField(line, 4, 8)
		if err != nil {
			return nil, fmt.Errorf("bad starting rank: %w", err)
		}

		if len(line) <= offset {
			continue // no data for the new round
		}

		// Opponent rank: 4 chars starting at offset.
		endPos := offset + 4
		if endPos > len(line) {
			endPos = len(line)
		}
		opponentStr := strings.TrimSpace(line[offset:endPos])

		opponent := 0
		if opponentStr != "" {
			opponent, err = strconv.Atoi(opponentStr)
			if err != nil {
				return nil, fmt.Errorf("invalid opponent rank for player %d: %q", rank, opponentStr)
			}
		}

		// Color: index offset+5.
		color := byte(' ')
		if len(line) > offset+5 {
			color = line[offset+5]
		}

		if opponent == 0 && color == '-' {
			// Bye (opponent 0 with explicit '-' color).
			boardNum++
			pairings = append(pairings, ParsedPairing{
				BoardNumber: boardNum,
				WhiteRank:   rank,
				BlackRank:   0,
			})
		} else if opponent == 0 {
			// Blank/no data for this round — player not paired, skip.
			continue
		} else if color == 'w' {
			// This player is white — record the pairing.
			boardNum++
			pairings = append(pairings, ParsedPairing{
				BoardNumber: boardNum,
				WhiteRank:   rank,
				BlackRank:   opponent,
			})
		}
		// color == 'b' is captured from the white side.
	}

	if len(pairings) == 0 {
		return nil, fmt.Errorf("no pairings found in output for round %d", newRound)
	}

	return pairings, nil
}

// ParseJaVaFoOutput parses the simple-pairs output format written by JaVaFo and
// by bbpPairings (both with -p):
//
//	Line 1:  number of pairings
//	Line 2+: "whiteRank blackRank" per line (blackRank 0 = bye)
func ParseJaVaFoOutput(output string) ([]ParsedPairing, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty output")
	}

	count, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid pair count on line 1: %w", err)
	}

	if len(lines) < count+1 {
		return nil, fmt.Errorf("expected %d pairing lines, got %d", count, len(lines)-1)
	}

	pairings := make([]ParsedPairing, 0, count)
	for i := 1; i <= count; i++ {
		parts := strings.Fields(lines[i])
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid pairing on line %d: %q", i+1, lines[i])
		}

		white, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid white rank on line %d: %w", i+1, err)
		}

		black, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid black rank on line %d: %w", i+1, err)
		}

		pairings = append(pairings, ParsedPairing{
			BoardNumber: i,
			WhiteRank:   white,
			BlackRank:   black,
		})
	}

	return pairings, nil
}

// parseIntField extracts an integer from a fixed-width field in a line.
// start and end are 0-indexed byte positions (end exclusive).
func parseIntField(line string, start, end int) (int, error) {
	if len(line) < end {
		return 0, fmt.Errorf("line too short: need %d chars, got %d", end, len(line))
	}
	s := strings.TrimSpace(line[start:end])
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}