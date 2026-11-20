package modals

import (
	"testing"
)

func FuzzMaskEnvVar(f *testing.F) {
	f.Add("KEY=value")
	f.Add("SECRET=mysecret")
	f.Add("DATABASE_URL=postgres://user:pass@localhost:5432/db")
	f.Add("NO_EQUALS_LINE")
	f.Add("=")
	f.Add("====")
	f.Add("A=B=C")
	f.Add("")

	f.Fuzz(func(_ *testing.T, entry string) {
		// Must never panic on arbitrary input
		_ = MaskEnvVar(entry)
	})
}
