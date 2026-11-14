// Package ui holds lith-ui, the TypeScript the site's controllers share
// (lith-ui.ts), and strips the types of the site's TypeScript for the
// browser. tsc checks the sources (go tool task typecheck); esbuild only
// drops their types, file by file, as the site starts, so no build step
// and no generated JavaScript sits in the repository.
package ui

import (
	_ "embed"
	"errors"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Source is lith-ui.ts. Modules import it as 'lith-ui', which the page's
// import map points at its build.
//
//go:embed lith-ui.ts
var Source string

// Transpile returns the TypeScript module src, named name in errors, as
// JavaScript for the browsers the site serves; minified unless dev.
// Imports stay as written, for the page's import map.
func Transpile(name, src string, dev bool) ([]byte, error) {
	r := api.Transform(src, api.TransformOptions{
		Loader:            api.LoaderTS,
		Format:            api.FormatESModule,
		Target:            api.ES2022,
		Sourcefile:        name,
		MinifyWhitespace:  !dev,
		MinifySyntax:      !dev,
		MinifyIdentifiers: !dev,
		Charset:           api.CharsetUTF8,
	})
	if len(r.Errors) > 0 {
		msgs := api.FormatMessages(r.Errors, api.FormatMessagesOptions{Kind: api.ErrorMessage})
		return nil, errors.New(strings.Join(msgs, "\n"))
	}
	return r.Code, nil
}
