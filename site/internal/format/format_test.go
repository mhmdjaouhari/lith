package format

import (
	"math"
	"testing"
)

func TestGrouped(t *testing.T) {
	for n, want := range map[int64]string{
		0: "0", 7: "7", 999: "999", 1000: "1,000", 12345: "12,345", 1_000_000: "1,000,000",
		-1: "-1", -999: "-999", -1000: "-1,000", -123456: "-123,456",
		math.MinInt64: "-9,223,372,036,854,775,808",
	} {
		if got := Grouped(n); got != want {
			t.Errorf("Grouped(%d) = %q, want %q", n, got, want)
		}
	}
	if got := string(AppendGrouped([]byte("#"), 4096)); got != "#4,096" {
		t.Errorf("AppendGrouped: %q", got)
	}
	if Plural(1, "entry", "entries") != "entry" || Plural(0, "entry", "entries") != "entries" {
		t.Error("Plural")
	}
}
