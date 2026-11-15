// Package ui holds lith-ui, the TypeScript the site's controllers share
// (lith-ui.ts), and builds the site's TypeScript for the browser. tsc
// checks the sources (go tool tsc); esbuild only drops their types and
// bundles a demo's modules, as the site starts, so no build step and no
// generated JavaScript sits in the repository.
package ui

import (
	_ "embed"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Source is lith-ui.ts. Modules import it as 'lith-ui', which the page's
// import map points at its build.
//
//go:embed lith-ui.ts
var Source string

// External are the modules the page's import map provides: a bundle
// imports them by name and never holds them.
var External = []string{"datastar", "lith-ui"}

func options(dev bool) (minify bool, target api.Target, format api.Format) {
	return !dev, api.ES2022, api.FormatESModule
}

// Transpile returns the TypeScript module src, named name in errors, as
// JavaScript for the browsers the site serves; minified unless dev.
// Imports stay as written, for the page's import map.
func Transpile(name, src string, dev bool) ([]byte, error) {
	minify, target, format := options(dev)
	r := api.Transform(src, api.TransformOptions{
		Loader:            api.LoaderTS,
		Format:            format,
		Target:            target,
		Sourcefile:        name,
		MinifyWhitespace:  minify,
		MinifySyntax:      minify,
		MinifyIdentifiers: minify,
		Charset:           api.CharsetUTF8,
	})
	if len(r.Errors) > 0 {
		return nil, errors.New(strings.Join(api.FormatMessages(r.Errors, api.FormatMessagesOptions{Kind: api.ErrorMessage}), "\n"))
	}
	return r.Code, nil
}

// Bundle returns the TypeScript module entry of fsys and the modules it
// imports from fsys by relative path (a demo's controller and the
// ds.gen.ts dsgen writes beside it) as one ES module, its types dropped;
// minified unless dev. The External modules stay imports.
func Bundle(fsys fs.FS, entry string, dev bool) ([]byte, error) {
	minify, target, format := options(dev)
	const ns = "fsys"
	plugin := api.Plugin{Name: "fsys", Setup: func(b api.PluginBuild) {
		b.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(a api.OnResolveArgs) (api.OnResolveResult, error) {
			for _, e := range External {
				if a.Path == e {
					return api.OnResolveResult{Path: a.Path, External: true}, nil
				}
			}
			p := a.Path
			if a.Kind != api.ResolveEntryPoint {
				if !strings.HasPrefix(p, "./") && !strings.HasPrefix(p, "../") {
					return api.OnResolveResult{}, errors.New("import " + p + ": a relative path, or one of the import map's modules")
				}
				p = path.Join(path.Dir(a.Importer), p)
			}
			if path.Ext(p) != ".ts" {
				p += ".ts"
			}
			return api.OnResolveResult{Path: path.Clean(p), Namespace: ns}, nil
		})
		b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: ns}, func(a api.OnLoadArgs) (api.OnLoadResult, error) {
			src, err := fs.ReadFile(fsys, a.Path)
			if err != nil {
				return api.OnLoadResult{}, err
			}
			s := string(src)
			return api.OnLoadResult{Contents: &s, Loader: api.LoaderTS}, nil
		})
	}}
	r := api.Build(api.BuildOptions{
		EntryPoints:       []string{entry},
		Bundle:            true,
		Write:             false,
		Outfile:           "out.js",
		Format:            format,
		Target:            target,
		MinifyWhitespace:  minify,
		MinifySyntax:      minify,
		MinifyIdentifiers: minify,
		Charset:           api.CharsetUTF8,
		Plugins:           []api.Plugin{plugin},
		LogLevel:          api.LogLevelSilent,
	})
	if len(r.Errors) > 0 {
		return nil, errors.New(strings.Join(api.FormatMessages(r.Errors, api.FormatMessagesOptions{Kind: api.ErrorMessage}), "\n"))
	}
	if len(r.OutputFiles) != 1 {
		return nil, errors.New("ui: " + entry + ": no single output")
	}
	return r.OutputFiles[0].Contents, nil
}
