package probe

import (
	"os"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/ebpf"
)

func TestRE2C(t *testing.T) {
	r := NewRE2C()

	re2cInput, err := r.patternsToInput([]string{"perdu.com"})
	if err != nil {
		t.Fatal(err)
	}

	outputFile, err := r.compile(re2cInput)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile)

	runtimeAsset, err := r.toEBPF(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	_, err = runtimeAsset.Compile(&ebpf.Config{
		EnableRuntimeCompiler:    true,
		RuntimeCompilerOutputDir: os.TempDir(),
		BPFDir:                   os.TempDir(),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
