package engine_test

import (
	"testing"
	"time"

	"github.com/tessechess/core/engine"
)

// The constructors return the concrete engine types (assigning to a typed
// variable fails to compile otherwise) and are equivalent to filling in the
// exported fields by hand.
func TestConstructors_ReturnConcreteTypes(t *testing.T) {
	var bbp *engine.BBPPairings = engine.NewBBPPairings("/opt/bbpPairings", 7*time.Second)
	if want := (engine.BBPPairings{BinaryPath: "/opt/bbpPairings", Timeout: 7 * time.Second}); *bbp != want {
		t.Errorf("NewBBPPairings = %+v, want %+v", *bbp, want)
	}
	if bbp.Name() != "bbppairings" || bbp.OutputFormat() != engine.OutputSimplePairs {
		t.Errorf("bbpPairings Name/OutputFormat = %q/%q", bbp.Name(), bbp.OutputFormat())
	}

	var jvf *engine.JaVaFo = engine.NewJaVaFo("java", "/opt/javafo.jar", 9*time.Second)
	if want := (engine.JaVaFo{JavaPath: "java", JarPath: "/opt/javafo.jar", Timeout: 9 * time.Second}); *jvf != want {
		t.Errorf("NewJaVaFo = %+v, want %+v", *jvf, want)
	}
	if jvf.Name() != "javafo" || jvf.OutputFormat() != engine.OutputSimplePairs {
		t.Errorf("javafo Name/OutputFormat = %q/%q", jvf.Name(), jvf.OutputFormat())
	}
}
