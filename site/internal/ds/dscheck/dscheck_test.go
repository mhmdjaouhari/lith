package dscheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type inner struct {
	Ok bool `json:"ok"`
}

type sigs struct {
	Op    string `json:"op"`
	Inner inner  `json:"inner"`
}

var schema = Schema{"_x": reflect.TypeFor[sigs]()}

func TestCheck(t *testing.T) {
	good := `<div data-signals='{"_x": {"op": "", "inner": {"ok": false}}}' data-ctl
		data-on:click__window="$_x.op = 'a'; @post('/cmd/x/y', {payload: {tab: $_tab}})"
		data-class:is-on="$_x.inner.ok" data-show="$_live" data-card="3" data-indicator:_busy>
		<p data-text="$_x.op"></p></div>`
	if errs := Check(good, schema, "ctl"); len(errs) > 0 {
		t.Fatalf("good page: %v", errs)
	}
	for name, c := range map[string]struct{ page, want string }{
		"misspelt signal":   {`<p data-text="$_x.pp"></p>`, `no signal "pp"`},
		"nested misspelt":   {`<p data-show="$_x.inner.okk"></p>`, `no signal "okk"`},
		"undeclared root":   {`<p data-show="$_y"></p>`, `declares no $_y`},
		"into a value":      {`<p data-show="$_x.op.length"></p>`, `without .length`},
		"syntax":            {`<p data-text="$_x.op +"></p>`, `not JavaScript`},
		"apostrophe":        {`<p data-on:click="$_x.op = 'it's'"></p>`, `not JavaScript`},
		"hyphenated key":    {`<p data-on-click="$_x.op = ''"></p>`, `data-on:click`},
		"hyphenated class":  {`<p data-class-on="$_x.inner.ok"></p>`, `data-class:on`},
		"misspelt plugin":   {`<p data-sho="$_x.op"></p>`, `no Datastar attribute`},
		"uppercase key":     {`<p data-on:Click="1"></p>`, ``}, // HTML lowercases it: fine
		"empty expression":  {`<p data-show=""></p>`, `no expression`},
		"controller value":  {`<p data-ctl="x"></p>`, `takes no key and no value`},
		"misspelt in a mod": {`<p data-on:click__window="$_x.opp"></p>`, `no signal "opp"`},
	} {
		errs := Check(c.page, schema, "ctl")
		switch {
		case c.want == "" && len(errs) > 0:
			t.Errorf("%s: %v", name, errs)
		case c.want != "" && (len(errs) != 1 || !strings.Contains(errs[0].Error(), c.want)):
			t.Errorf("%s: %v, want one error with %q", name, errs, c.want)
		}
	}
}

func TestLint(t *testing.T) {
	dir := t.TempDir()
	src := "templ X() {\n\t<p data-show=\"$_x.op\" data-text={ s }></p>\n\t<p data-card=\"3\" data-on:click={ x }></p>\n\t<p data-on:click__window=\"@post('/x')\"></p>\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "x.templ"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	errs, err := Lint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 2 || !strings.Contains(errs[0].Error(), "x.templ:2: data-show") || !strings.Contains(errs[1].Error(), "x.templ:4: data-on") {
		t.Errorf("%v", errs)
	}
}
