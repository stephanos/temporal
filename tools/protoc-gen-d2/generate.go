package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
	d2log "oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
	"oss.terrastruct.com/util-go/go2"
)

func generateFile(gen *protogen.Plugin, f *protogen.File) {
	buf := new(bytes.Buffer)

	if err := tmpl.Execute(buf, f); err != nil {
		gen.Error(err)
		return
	}
	gf := gen.NewGeneratedFile(f.GeneratedFilenamePrefix+".d2", f.GoImportPath)
	if _, err := gf.Write(buf.Bytes()); err != nil {
		gen.Error(err)
		return
	}

	svgf := gen.NewGeneratedFile(f.GeneratedFilenamePrefix+".svg", f.GoImportPath)
	svg, err := renderSvg(buf.String())
	if err != nil {
		gen.Error(err)
		return
	}
	if _, err := svgf.Write(svg); err != nil {
		gen.Error(err)
		return
	}
}

func renderSvg(d2text string) ([]byte, error) {
	ruler, _ := textmeasure.NewRuler()
	layoutResolver := func(engine string) (d2graph.LayoutGraph, error) {
		return d2dagrelayout.DefaultLayout, nil
	}
	renderOpts := &d2svg.RenderOpts{
		Pad:     go2.Pointer(int64(5)),
		ThemeID: &d2themescatalog.Aubergine.ID,
		Scale:   go2.Pointer(0.7),
	}
	compileOpts := &d2lib.CompileOptions{
		LayoutResolver: layoutResolver,
		Ruler:          ruler,
	}
	ctx := d2log.With(context.Background(), slog.New(slog.NewTextHandler(os.Stdout, nil)))
	diagram, _, err := d2lib.Compile(ctx, d2text, compileOpts, renderOpts)
	if err != nil {
		return nil, fmt.Errorf("d2 source: %v \n\nrendering error: %w", d2text, err)
	}
	return d2svg.Render(diagram, renderOpts)
}
