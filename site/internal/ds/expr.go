package ds

import (
	"strconv"
	"time"
)

// Small pieces of Datastar expressions and attributes that every page
// writes the same way. Each takes an expression (a Ref's String, or one
// built from refs) and returns one.

// Aria is an ARIA state that is true or false, aria-pressed say:
// 'true' while cond holds, else 'false'.
func Aria(cond string) string { return cond + " ? 'true' : 'false'" }

// AriaBusy is aria-busy while cond holds. data-attr writes true as an
// empty value, which aria-busy reads as false, and false removes the
// attribute: so 'true', or false.
func AriaBusy(cond string) string { return cond + " ? 'true' : false" }

// Trimmed is v, a string, without its spaces at the ends. The parens keep
// .trim() out of the signal's path, which dscheck reads up to the last
// dot.
func Trimmed(v string) string { return "(" + v + ").trim()" }

// Ms is d as a Datastar modifier's duration: data-on:input__throttle.
// + Ms(100*time.Millisecond) is data-on:input__throttle.100ms.
func Ms(d time.Duration) string { return strconv.FormatInt(d.Milliseconds(), 10) + "ms" }
