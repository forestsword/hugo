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

package asciidoc

import (
	"testing"

	"github.com/gohugoio/hugo/common/loggers"

	"github.com/gohugoio/hugo/markup/converter"

	qt "github.com/frankban/quicktest"
)

func TestConvert(t *testing.T) {
	if !Supports() {
		t.Skip("asciidoc/asciidoctor not installed")
	}
	c := qt.New(t)
	p, err := Provider.New(converter.ProviderConfig{Logger: loggers.NewErrorLogger()})
	c.Assert(err, qt.IsNil)
	conv, err := p.New(converter.DocumentContext{})
	c.Assert(err, qt.IsNil)
	b, err := conv.Convert(converter.RenderContext{Src: []byte("testContent")})
	c.Assert(err, qt.IsNil)
	c.Assert(string(b.Bytes()), qt.Equals, "<div class=\"paragraph\">\n<p>testContent</p>\n</div>\n")
}

func TestTableOfContents(t *testing.T) {
	if !Supports() {
		t.Skip("asciidoc/asciidoctor not installed")
	}
	c := qt.New(t)
	p, err := Provider.New(converter.ProviderConfig{Logger: loggers.NewErrorLogger()})
	c.Assert(err, qt.IsNil)
	conv, err := p.New(converter.DocumentContext{})
	c.Assert(err, qt.IsNil)
	b, err := conv.Convert(converter.RenderContext{Src: []byte(`:toc: macro
:toclevels: 4
toc::[]

=== Introduction

== Section 1

=== Section 1.1

==== Section 1.1.1

=== Section 1.2

testContent

== Section 2
`)})
	c.Assert(err, qt.IsNil)
	toc, ok := b.(converter.TableOfContentsProvider)
	c.Assert(ok, qt.Equals, true)
	root := toc.TableOfContents()
	c.Assert(root.ToHTML(2, 4, false), qt.Equals, "<nav id=\"TableOfContents\">\n  <ul>\n    <li><a href=\"#_introduction\">Introduction</a></li>\n    <li><a href=\"#_section_1\">Section 1</a>\n      <ul>\n        <li><a href=\"#_section_1_1\">Section 1.1</a>\n          <ul>\n            <li><a href=\"#_section_1_1_1\">Section 1.1.1</a></li>\n          </ul>\n        </li>\n        <li><a href=\"#_section_1_2\">Section 1.2</a></li>\n      </ul>\n    </li>\n    <li><a href=\"#_section_2\">Section 2</a></li>\n  </ul>\n</nav>")
	c.Assert(root.ToHTML(2, 3, false), qt.Equals, "<nav id=\"TableOfContents\">\n  <ul>\n    <li><a href=\"#_introduction\">Introduction</a></li>\n    <li><a href=\"#_section_1\">Section 1</a>\n      <ul>\n        <li><a href=\"#_section_1_1\">Section 1.1</a></li>\n        <li><a href=\"#_section_1_2\">Section 1.2</a></li>\n      </ul>\n    </li>\n    <li><a href=\"#_section_2\">Section 2</a></li>\n  </ul>\n</nav>")
}
