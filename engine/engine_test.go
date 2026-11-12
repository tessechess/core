//go:build unix

// The tests run #!/bin/sh mock engines, so they only run on Unix-like systems.

package engine_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tessechess/core/engine"
	"github.com/tessechess/core/trf"
)

// writeMockScript creates an executable shell script in a temp dir and returns its path.
func writeMockScript(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}
	return path
}

const sampleTRF = `012 Test Tournament
XXR 3
001    1 m    Carlsen, Magnus                  2830 NOR     1503014 1990/11/30  0.0
001    2 m    Caruana, Fabiano                  2800 USA     2020009 1992/07/30  0.0
`

func TestBBPPairings_Success(t *testing.T) {
	// Mock bbpPairings: like the real binary (v6.0.0), -p writes the simple pair
	// list ("count\nwhite black\n..."), not a TRF.
	mock := writeMockScript(t, "bbpPairings", `
OUTPUT=""
while [ $# -gt 0 ]; do
    case "$1" in
        --dutch) shift ;;
        -p) OUTPUT="$2"; shift 2 ;;
        *) shift ;;
    esac
done
printf "1\n1 2\n" > "$OUTPUT"
`)

	eng := engine.NewBBPPairings(mock, 10*time.Second)

	if eng.Name() != "bbppairings" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "bbppairings")
	}
	if eng.OutputFormat() != engine.OutputSimplePairs {
		t.Errorf("OutputFormat() = %q, want %q", eng.OutputFormat(), engine.OutputSimplePairs)
	}

	output, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err != nil {
		t.Fatalf("GeneratePairings error: %v", err)
	}

	parsed, err := trf.ParseSimplePairs(output)
	if err != nil {
		t.Fatalf("output is not a simple pair list: %v (output %q)", err, output)
	}
	if len(parsed) != 1 || parsed[0].WhiteRank != 1 || parsed[0].BlackRank != 2 {
		t.Errorf("parsed = %+v, want one board 1 vs 2", parsed)
	}
}

func TestJaVaFo_Success(t *testing.T) {
	// Mock JaVaFo: ignores input, writes simple pairs to output file.
	// Args: -jar <jar> <input> -p <output>
	mock := writeMockScript(t, "java", `
OUTPUT=""
INPUT=""
skip_next=false
for arg in "$@"; do
    if $skip_next; then
        skip_next=false
        continue
    fi
    case "$arg" in
        -jar) skip_next=true ;;
        -p) skip_next=true; OUTPUT="$1" ;;
    esac
    shift 2>/dev/null
done
# Re-parse: -jar X input -p output
set -- $0  # reset
OUTPUT=""
INPUT=""
i=0
for arg in '"$@"'; do :; done
# Simpler: just find -p and the arg after it
args="$*"
OUTPUT=$(echo "$args" | sed -n 's/.*-p \([^ ]*\).*/\1/p')
echo "1" > "$OUTPUT"
echo "1 2" >> "$OUTPUT"
`)

	// The mock script arg parsing is tricky with sh. Let's use a simpler approach:
	mock = writeMockScript(t, "java", `
# Find output path: last argument after -p
OUTPUT=""
while [ $# -gt 0 ]; do
    if [ "$1" = "-p" ]; then
        OUTPUT="$2"
        shift 2
    else
        shift
    fi
done
echo "1" > "$OUTPUT"
echo "1 2" >> "$OUTPUT"
`)

	// Create a fake jar file so the Stat check passes.
	jarDir := t.TempDir()
	jarPath := filepath.Join(jarDir, "javafo.jar")
	if err := os.WriteFile(jarPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	eng := engine.NewJaVaFo(mock, jarPath, 10*time.Second)

	if eng.Name() != "javafo" {
		t.Errorf("Name() = %q, want %q", eng.Name(), "javafo")
	}
	if eng.OutputFormat() != engine.OutputSimplePairs {
		t.Errorf("OutputFormat() = %q, want %q", eng.OutputFormat(), engine.OutputSimplePairs)
	}

	output, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err != nil {
		t.Fatalf("GeneratePairings error: %v", err)
	}

	if output == "" {
		t.Fatal("expected non-empty output")
	}

	if !containsSubstring(output, "1 2") {
		t.Errorf("output missing expected pairing, got: %q", output)
	}
}

func TestEngine_BinaryNotFound(t *testing.T) {
	eng := engine.NewBBPPairings("/nonexistent/bbpPairings", 5*time.Second)
	_, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !errors.Is(err, engine.ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got: %v", err)
	}
}

func TestEngine_JaVaFo_JarNotFound(t *testing.T) {
	eng := engine.NewJaVaFo("java", "/nonexistent/javafo.jar", 5*time.Second)
	_, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err == nil {
		t.Fatal("expected error for missing jar")
	}
	if !errors.Is(err, engine.ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got: %v", err)
	}
}

func TestEngine_JaVaFo_JavaNotFound(t *testing.T) {
	// The jar exists, the java executable does not.
	jarPath := filepath.Join(t.TempDir(), "javafo.jar")
	if err := os.WriteFile(jarPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	eng := engine.NewJaVaFo("/nonexistent/java", jarPath, 5*time.Second)
	_, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err == nil {
		t.Fatal("expected error for missing java")
	}
	if !errors.Is(err, engine.ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got: %v", err)
	}
	if !containsSubstring(err.Error(), "/nonexistent/java") {
		t.Errorf("error should name the java path, got: %v", err)
	}
}

func TestEngine_Timeout(t *testing.T) {
	mock := writeMockScript(t, "slow-engine", "sleep 30 &\nwait")
	eng := engine.NewBBPPairings(mock, 200*time.Millisecond)

	_, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, engine.ErrEngineTimeout) {
		t.Errorf("expected ErrEngineTimeout, got: %v", err)
	}
}

func TestEngine_NonZeroExit(t *testing.T) {
	mock := writeMockScript(t, "failing-engine", `echo "something went wrong" >&2; exit 1`)
	eng := engine.NewBBPPairings(mock, 5*time.Second)

	_, err := eng.GeneratePairings(context.Background(), sampleTRF)
	if err == nil {
		t.Fatal("expected execution error")
	}
	if !errors.Is(err, engine.ErrEngineExecution) {
		t.Errorf("expected ErrEngineExecution, got: %v", err)
	}
	if !containsSubstring(err.Error(), "something went wrong") {
		t.Errorf("error should contain stderr, got: %v", err)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
