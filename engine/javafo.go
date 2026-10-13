package engine

import (
	"context"
	"fmt"
	"os"
	"time"
)

// JaVaFo implements PairingEngine for the JaVaFo jar.
type JaVaFo struct {
	JavaPath string
	JarPath  string
	Timeout  time.Duration
}

func NewJaVaFo(javaPath, jarPath string, timeout time.Duration) PairingEngine {
	return &JaVaFo{
		JavaPath: javaPath,
		JarPath:  jarPath,
		Timeout:  timeout,
	}
}

func (j *JaVaFo) Name() string { return "javafo" }

func (j *JaVaFo) OutputFormat() OutputFormat { return OutputSimplePairs }

func (j *JaVaFo) GeneratePairings(ctx context.Context, trfInput string) (string, error) {
	if _, err := os.Stat(j.JarPath); err != nil {
		return "", fmt.Errorf("%w: %s", ErrEngineNotFound, j.JarPath)
	}

	ctx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()

	return RunEngine(ctx, j.JavaPath, func(inputPath, outputPath string) []string {
		return []string{"-jar", j.JarPath, inputPath, "-p", outputPath}
	}, trfInput, true)
}
