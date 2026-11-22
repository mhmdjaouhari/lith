package ds

import (
	"testing"
	"time"
)

func TestExpr(t *testing.T) {
	op := NewRef[string]("_x.op")
	ack := NewRef[string]("_x.ack")
	n := NewRef[int]("_x.n")
	on := NewRef[bool]("_x.on")
	for _, c := range []struct{ got, want Expr }{
		{Expr(op), `$_x.op`},
		{op.Eq("a"), `$_x.op === "a"`},
		{op.Assign(`it's`), `$_x.op = "it's"`},
		{n.Ge(0), `$_x.n >= 0`},
		{n.Inc(), `$_x.n = $_x.n + 1`},
		{ack.AtLeast(op), `$_x.ack >= $_x.op`},
		{op.To(OrderedOpID), `$_x.op = @orderedOpId()`},
		{Not(on), `!$_x.on`},
		{Not(op.Eq("")), `!($_x.op === "")`},
		{And(op.Ne(""), ack.Is(op)), `($_x.op !== "") && ($_x.ack === $_x.op)`},
		{Or(on, Val(true)), `$_x.on || true`},
		{If(on, Call("go")), `$_x.on && @go()`},
		{If(on, op.Assign(""), on.Assign(false)), `$_x.on && ($_x.op = "", $_x.on = false)`},
		{Do(Call("a", n, Val("b")), Call("c")), `@a($_x.n, "b"), @c()`},
		{Concat(op, ack), `$_x.op + $_x.ack`},
		{Aria(op.Eq("a")), `($_x.op === "a") ? 'true' : 'false'`},
		{AriaBusy(on), `$_x.on ? 'true' : false`},
		{Trimmed(op), `($_x.op).trim()`},
		{n.Gt(0), `$_x.n > 0`},
		{n.Le(1), `$_x.n <= 1`},
		{Cond(n.Ge(3), Val("KB"), op), `($_x.n >= 3) ? "KB" : $_x.op`},
		{Max(Val(0), n), `Math.max(0, $_x.n)`},
		{If(on, Invoke("push", n)), `$_x.on && (el.push && el.push($_x.n))`},
	} {
		if c.got != c.want {
			t.Errorf("got %s, want %s", c.got, c.want)
		}
	}
	if op.Path() != "_x.op" || op.String() != "$_x.op" {
		t.Errorf("path %s, string %s", op.Path(), op.String())
	}
	for _, d := range []struct {
		d    time.Duration
		want string
	}{{1500 * time.Microsecond, "1ms"}, {2 * time.Second, "2000ms"}} {
		if got := Ms(d.d); got != d.want {
			t.Errorf("Ms(%s) = %s, want %s", d.d, got, d.want)
		}
	}
}

func TestCallName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a dotted action name did not panic")
		}
	}()
	Call("life.drop")
}
