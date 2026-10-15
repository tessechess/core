package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var _ PairingEngine = (*JaVaFo)(nil)

// JaVaFo is the PairingEngine for the JaVaFo jar. It runs
// "java -jar <jar> <input> -p <output>" (pair list output).
//
// Build one with NewJaVaFo, or fill in the fields directly; Timeout must then
// be set, because a zero Timeout makes every run time out immediately.
type JaVaFo struct {
	JavaPath string        // the java executable ("java" to use the one on PATH)
	JarPath  string        // path to the JaVaFo jar
	Timeout  time.Duration // longest a single run may take; see the package docs on child processes
}

// NewJaVaFo returns an engine that runs the jar at jarPath with the Java
// executable javaPath, killing it after timeout. Nothing is checked until
// GeneratePairings is called.
func NewJaVaFo(javaPath, jarPath string, timeout time.Duration) *JaVaFo {
	return &JaVaFo{
		JavaPath: javaPath,
		JarPath:  jarPath,
		Timeout:  timeout,
	}
}

// Name returns "javafo".
func (j *JaVaFo) Name() string { return "javafo" }

// OutputFormat is OutputSimplePairs: with -p, JaVaFo writes the pair list
// ("count\nwhite black\n...").
func (j *JaVaFo) OutputFormat() OutputFormat { return OutputSimplePairs }

// GeneratePairings pairs the next round of the TRF in trfInput and returns the
// engine's pair list. It returns ErrEngineNotFound if JarPath does not
// exist or JavaPath is not an executable (looked up on PATH if it has no
// directory part), ErrEngineTimeout if the run exceeds Timeout (or the deadline of ctx
// passes first) and ErrEngineExecution, with the engine's output, if it exits
// with an error.
func (j *JaVaFo) GeneratePairings(ctx context.Context, trfInput string) (string, error) {
	if _, err := os.Stat(j.JarPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrEngineNotFound, j.JarPath)
	}

	javaPath, err := exec.LookPath(j.JavaPath)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrEngineNotFound, j.JavaPath)
	}

	ctx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()

	return runEngine(ctx, javaPath, func(inputPath, outputPath string) []string {
		return []string{"-jar", j.JarPath, inputPath, "-p", outputPath}
	}, trfInput)
}
