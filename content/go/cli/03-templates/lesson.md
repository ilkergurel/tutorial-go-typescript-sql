---
id: go.cli.templates
title: "text/template vs html/template"
track: go
module: cli
order: 3
level: intermediate
minutes: 14
since: go1.18
prerequisites: [go.cli.fmt, go.composite.structs]
compare_with: [ts.react.components, design.architecture.security]
tags: [templates, text-template, html-template, xss, escaping]
summary: >
  text/template and html/template share one template language and one API.
  html/template adds escaping that depends on where each value lands (HTML,
  attribute, URL, JavaScript), which is what stops XSS.
examples:
  - id: text-template
    file: examples/01_text_template.go
    title: "A text report: range, else, with, methods and a FuncMap"
    runtime: go
    check: exact
    annotations:
      - lines: "25-33"
        note: >
          {{range}} and {{with}} rebind the dot: to each Item, then to the Note.
          {{else}} runs when the slice is empty, and {{with}} skips its body for
          an empty Note. A dash inside the braces ({{- or -}}) trims the
          whitespace on that side, which is how a text template keeps control
          over its newlines.
      - lines: "27"
        note: >
          .Total calls the method on Item, and printf is a built-in function with
          fmt's verbs. There is no arithmetic in the language: anything beyond
          comparisons is a method or a function you register.
      - lines: "36-39"
        note: >
          Functions must be registered before Parse, which otherwise fails with
          function "upper" not defined. Must turns a parse error into a panic,
          the right reaction to a broken template constant.
      - lines: "52-57"
        note: >
          Field names are resolved at Execute time, by reflection. The typo
          compiles, parses, and fails halfway through the output. Render into a
          buffer when a partial result would hurt, like a half-written HTTP response.
  - id: html-escaping
    file: examples/02_html_escaping.go
    title: The same template through both packages
    runtime: go
    check: exact
    annotations:
      - lines: "13-17"
        note: >
          One value in five contexts. Each needs its own escaping: entities in
          the body and in attributes, percent-encoding in a query string,
          JavaScript string syntax inside script, and a scheme check in href.
      - lines: "28-29"
        note: >
          text/template copies the bytes, so the quote closes the attribute and
          the script tag runs. html/template turns the javascript: URL into
          #ZgotmplZ, and percent-encodes ş as %c5%9f, its two UTF-8 bytes.
      - lines: "31-37"
        note: >
          Both types have the same constructors and Execute method, which makes
          switching packages a one-line import change, and makes picking the
          wrong one easy to miss.
  - id: layout-trusted
    file: examples/03_layout_trusted.go
    title: "Layouts with block and Clone; template.HTML for trusted markup"
    runtime: go
    check: exact
    annotations:
      - lines: "10-15"
        note: >
          {{block}} defines a named template and calls it in place, so the
          layout renders on its own with the defaults. A page redefines the
          blocks it needs with {{define}}.
      - lines: "17-21"
        note: >
          The field's type decides the escaping. A string is escaped; a
          template.HTML is trusted and inserted as is. Use it only for markup
          your code produced or a sanitizer returned.
      - lines: "24-27"
        note: >
          Clone copies the set of named templates, so each page gets its own
          title and content while base stays untouched. Clone returns an error
          once the template has been executed: build all pages at startup.
      - lines: "40-44"
        note: >
          Converting user input to template.HTML disables the protection for
          that value: the onerror handler is now live in the page.
use_cases:
  - "text/template: generated Go code (go generate), config files, plain-text emails, reports"
  - "text/template: user-chosen output formats, the --format '{{.Name}}' flag several Go CLIs offer"
  - "html/template: server-rendered pages, HTML emails, admin screens without a frontend build"
strengths:
  - "Escaping follows the context: html/template knows whether {{.}} sits in a body, an attribute, a URL or a script. Jinja2 or Handlebars autoescaping applies one HTML-entity escape everywhere, which does not make a javascript: URL or a value inside script safe"
  - "One template language and one API for both packages: learn it once, switch the import for HTML"
  - "Standard library, no build step; with embed.FS and ParseFS (Go 1.16) the templates ship inside the binary"
  - "Deliberately small language (no assignment to fields, no arithmetic), which keeps business logic in Go code"
weaknesses:
  - "No compile-time checking: a misspelled field fails at Execute, while TSX checks every prop at compile time"
  - "Minimal expressions: no + or -, so even an index+1 needs a registered function"
  - "Typed strings (template.HTML, template.URL, template.JS) switch escaping off wholesale; a careless conversion reintroduces XSS"
  - "Whitespace control with {{- and -}} is fiddly when the output is plain text"
pitfalls:
  - title: "Importing text/template for HTML"
    detail: "The two packages have the same API, so the compiler can't tell you picked the wrong one, and an editor's auto-import may choose either. Check the import in every file that renders HTML."
    bad: "import \"text/template\" // in an HTTP handler"
    good: "import \"html/template\""
  - title: "template.HTML(userInput)"
    detail: "The conversion tells html/template the value is safe. Use typed strings only for constants, your own templates' output or a sanitizer's output."
  - title: "Partial output on error"
    detail: "Execute writes as it goes. In an HTTP handler, a failure halfway means a half page with status 200 already sent. Execute into a bytes.Buffer, then write it."
  - title: "Missing map keys print <no value>"
    detail: "With map data, text/template prints <no value> for an absent key instead of failing. Set t.Option(\"missingkey=error\") to make it an error."
  - title: "Unexported fields"
    detail: "Templates use reflection and can't read lowercase fields: Execute fails with \"name is an unexported field of struct type\"."
quiz:
  - q: "html/template renders <a href=\"{{.}}\"> with the value javascript:alert(1). What comes out?"
    options: ["The URL, HTML-escaped", "href=\"#ZgotmplZ\"", "An Execute error", "The value unchanged"]
    answer: 1
    why: "In a URL attribute, html/template only allows safe schemes; anything else is replaced by the marker #ZgotmplZ."
  - q: "A template contains {{.Custmer}} (typo) and the data is a struct. When does the problem surface?"
    options: ["At compile time", "At Parse", "At Execute"]
    answer: 2
    why: "Fields are resolved by reflection while executing; Parse only checks syntax and function names."
  - q: "When is converting a string to template.HTML appropriate?"
    options: ["For a user's bio that contains formatting", "For markup your own code built or a sanitizer returned", "Whenever the string contains tags"]
    answer: 1
    why: "template.HTML is inserted verbatim. It must never carry untrusted input."
  - q: "You switch a working program from text/template to html/template. What usually has to change besides the import?"
    options: ["Every action needs | html", "The templates need a new syntax", "Usually nothing: the package adds the escaping itself"]
    answer: 2
    why: "The packages share the template language and API; html/template adds the right escapers to each action by itself."
further_reading:
  - title: "package text/template"
    url: "https://pkg.go.dev/text/template"
  - title: "package html/template"
    url: "https://pkg.go.dev/html/template"
  - title: "MDN: Cross-site scripting (XSS)"
    url: "https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/XSS"
---

## Concept

`text/template` is a small template language: `{{.Field}}` prints data, `{{range}}`,
`{{if}}` and `{{with}}` control the flow, pipelines pass values through functions like
shell pipes. Data comes in as any Go value and is read by reflection at execution time.

`html/template` wraps the same engine. It works out, for every action, where the
value will land (element text, attribute, URL, CSS, JavaScript) and inserts the
matching escaper. The template author is trusted; the data passed to `Execute` is
not. That is the whole defence against cross-site scripting, and it costs one import
line.

## Coming from C/C++/Java/Python/JS

- Python: Jinja2 has more expression power (arithmetic, filters, inheritance with
  `extends`). Go's `{{block}}` plus `Clone` covers layouts with less. Jinja2's
  autoescaping is one HTML-entity escape; Go's is context-aware.
- Java: Thymeleaf escapes `th:text` by default; `template.HTML` plays the role of
  its unescaped `th:utext`.
- JS/React: JSX escapes text children much as html/template does, and
  `dangerouslySetInnerHTML` is the analogue of `template.HTML`. Template literals
  with string concatenation are the analogue of text/template: no escaping at all.
- C: think of text/template as a typed, reflective `printf` over whole documents.

## Design notes

Parse all templates once at startup, with `template.Must`, so a broken template stops
the program before it serves anything. Keep logic in Go: pass a view struct that
already holds the formatted values, rather than registering many functions. For a
site, put the layout and the pages in files, embed them with `embed.FS` and parse
them with `ParseFS`, cloning the layout per page as in the third example.

Use `text/template` only for output that is not HTML: code, configs, plain-text mail.
Anything a browser will render, including an "internal" admin page, goes through
`html/template`, and every `template.HTML` conversion deserves a code-review comment
saying where the markup came from.
