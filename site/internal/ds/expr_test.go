package ds

import (
	"testing"
	"time"
)

func TestExpr(t *testing.T) {
	op := NewRef[string]("_x.op")
	for got, want := range map[string]string{
		Aria(op.Eq("a")):            `$_x.op === "a" ? 'true' : 'false'`,
		AriaBusy(op.Ne("")):         `$_x.op !== "" ? 'true' : false`,
		Trimmed(op.String()):        `($_x.op).trim()`,
		Ms(1500 * time.Microsecond): `1ms`,
		Ms(2 * time.Second):         `2000ms`,
	} {
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}
