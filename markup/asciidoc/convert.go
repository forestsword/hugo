// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package asciidoc converts Asciidoc to HTML using Asciidoc or Asciidoctor
// external binaries.
package asciidoc

import (
	"bytes"
	"os/exec"

	"github.com/gohugoio/hugo/identity"
	"github.com/gohugoio/hugo/markup/converter"
	"github.com/gohugoio/hugo/markup/internal"
	"github.com/gohugoio/hugo/markup/tableofcontents"
	"golang.org/x/net/html"
)

// Provider is the package entry point.
var Provider converter.ProviderProvider = provider{}

type provider struct {
}

func (p provider) New(cfg converter.ProviderConfig) (converter.Provider, error) {
	return converter.NewProvider("asciidoc", func(ctx converter.DocumentContext) (converter.Converter, error) {
		return &asciidocConverter{
			ctx: ctx,
			cfg: cfg,
		}, nil
	}), nil
}

type asciidocResult struct {
	converter.Result
	toc tableofcontents.Root
}

func (r asciidocResult) TableOfContents() tableofcontents.Root {
	return r.toc
}

type asciidocConverter struct {
	ctx converter.DocumentContext
	cfg converter.ProviderConfig
}

func (a *asciidocConverter) Convert(ctx converter.RenderContext) (converter.Result, error) {
	content, toc, err := extractTOC(a.getAsciidocContent(ctx.Src, a.ctx))
	if err != nil {
		return nil, err
	}
	return asciidocResult{
		Result: converter.Bytes(content),
		toc:    toc,
	}, nil
}

func (a *asciidocConverter) Supports(_ identity.Identity) bool {
	return false
}

// getAsciidocContent calls asciidoctor or asciidoc as an external helper
// to convert AsciiDoc content to HTML.
func (a *asciidocConverter) getAsciidocContent(src []byte, ctx converter.DocumentContext) []byte {
	var isAsciidoctor bool
	path := getAsciidoctorExecPath()
	if path == "" {
		path = getAsciidocExecPath()
		if path == "" {
			a.cfg.Logger.ERROR.Println("asciidoctor / asciidoc not found in $PATH: Please install.\n",
				"                 Leaving AsciiDoc content unrendered.")
			return src
		}
	} else {
		isAsciidoctor = true
	}

	a.cfg.Logger.INFO.Println("Rendering", ctx.DocumentName, "with", path, "...")
	args := []string{"--no-header-footer", "--safe"}
	if isAsciidoctor {
		// asciidoctor-specific arg to show stack traces on errors
		args = append(args, "--trace")
	}
	args = append(args, "-")
	return internal.ExternallyRenderContent(a.cfg, ctx, src, path, args)
}

func getAsciidocExecPath() string {
	path, err := exec.LookPath("asciidoc")
	if err != nil {
		return ""
	}
	return path
}

func getAsciidoctorExecPath() string {
	path, err := exec.LookPath("asciidoctor")
	if err != nil {
		return ""
	}
	return path
}

// extractTOC extracts the toc from the given src html.
// It returns the html without the TOC, and the TOC data
func extractTOC(src []byte) ([]byte, tableofcontents.Root, error) {
	var buf bytes.Buffer
	buf.Write(src)
	node, err := html.Parse(&buf)
	if err != nil {
		return nil, tableofcontents.Root{}, err
	}
	var (
		f       func(*html.Node) bool
		toc     tableofcontents.Root
		toVisit []*html.Node
	)
	f = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "toc" {
					toc, err = parseTOC(n)
					if err != nil {
						return false
					}
					n.Parent.RemoveChild(n)
					return true
				}
			}
		}
		if n.FirstChild != nil {
			toVisit = append(toVisit, n.FirstChild)
		}
		if n.NextSibling != nil {
			if ok := f(n.NextSibling); ok {
				return true
			}
		}
		for len(toVisit) > 0 {
			nv := toVisit[0]
			toVisit = toVisit[1:]
			if ok := f(nv); ok {
				return true
			}
		}
		return false
	}
	f(node)
	if err != nil {
		return nil, tableofcontents.Root{}, err
	}
	buf.Reset()
	err = html.Render(&buf, node)
	if err != nil {
		return nil, tableofcontents.Root{}, err
	}
	// ltrim <html><head></head><body> and rtrim </body></html> which are added by html.Render
	res := buf.Bytes()[25:]
	res = res[:len(res)-14]
	return res, toc, nil
}

// parseTOC returns a TOC root from the given toc Node
func parseTOC(doc *html.Node) (tableofcontents.Root, error) {
	var (
		toc tableofcontents.Root
		f   func(*html.Node, int, int)
	)
	f = func(n *html.Node, parent, level int) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "ul":
				if level == 0 {
					parent += 1
				}
				level += 1
				f(n.FirstChild, parent, level)
			case "li":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type != html.ElementNode || c.Data != "a" {
						continue
					}
					var href string
					for _, a := range c.Attr {
						if a.Key == "href" {
							href = a.Val[1:]
							break
						}
					}
					for d := c.FirstChild; d != nil; d = d.NextSibling {
						if d.Type == html.TextNode {
							toc.AddAt(tableofcontents.Header{
								Text: d.Data,
								ID:   href,
							}, parent, level)
						}
					}
				}
				f(n.FirstChild, parent, level)
			}
		}
		if n.NextSibling != nil {
			f(n.NextSibling, parent, level)
		}
	}
	f(doc.FirstChild, 0, 0)
	return toc, nil
}

// Supports returns whether Asciidoc or Asciidoctor is installed on this computer.
func Supports() bool {
	return (getAsciidoctorExecPath() != "" ||
		getAsciidocExecPath() != "")
}
