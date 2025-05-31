package main

import "C"
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

var (
	lang        = ts.NewLanguage(golang.Language())
	parser      = ts.NewParser()
	query, qerr = ts.NewQuery(lang, goQuery)
)

func main() {
	if qerr != nil {
		panic(qerr)
	}

	args := []string{"doc"}
	args = append(args, os.Args[1:]...)
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\x1b[91;1mERROR:\x1b[0m %s\n\n%s", err.Error(), out)
		os.Exit(1)
		return
	}

	parser.SetLanguage(lang)
	lines := fixupSyntax(string(out))
	block := ""
	// Go doc isn't syntactically correct, so after some time,
	// TreeSitter breaks & doesn't highlight properly
	for _, line := range lines {
		block += line + "\n"
		if line == "" {
			printTree([]byte(block))
			block = ""
		}
	}
}

func printTree(block []byte) {
	// Nothing will ever highlight properly if this is removed.
	// TreeSitter depends on the package statement for whatever reason,
	// but this is obviously removed before printing.
	block = []byte("package main\n\n" + string(block))
	tree := parser.Parse(block, nil)

	cursor := ts.NewQueryCursor()
	captures := cursor.Captures(query, tree.RootNode(), block)

	chunks := []*[]*Chunk{}
	var lastChunk *[]*Chunk
	lastToken := &Chunk{&Highlight{}, ""}
	lastByte := uint(0)
	hls := query.CaptureNames()
	line := ""

	for {
		matches, _ := captures.Next()
		if matches == nil {
			break
		}
		for _, capture := range matches.Captures {
			id := capture.Index
			node := capture.Node

			start := node.StartPosition()
			startByte, endByte := node.StartByte(), node.EndByte()
			for start.Row >= uint(len(chunks)-1) || lastChunk == nil {
				line = ""
				lastChunk = &[]*Chunk{}
				chunks = append(chunks, lastChunk)
			}

			var hl string
			if 0 <= id && id < uint32(len(hls)) {
				hl = "@" + hls[id]
			}

			if startByte == lastByte {
				(*lastToken).Hl.Merge(highlights[hl])
			} else {
				diff := start.Column - uint(len(line))
				highlight, ok := highlights[hl]
				if ok {
					highlight = Highlight{highlight.Color, highlight.Bold, highlight.Ital}
				}
				lastToken = &Chunk{
					Text: string(block[startByte-diff:startByte]) + string(block[startByte:endByte]),
					Hl:   &highlight,
				}
				*lastChunk = append(*lastChunk, lastToken)
				line = line + lastToken.Text
				lastByte = startByte
			}

		}
	}

	for _, line := range chunks[2:] {
		for _, chunk := range *line {
			fmt.Print((*chunk).Hl.Wrap((*chunk).Text))
		}
		fmt.Println()
	}
}

type Sakura int
type Tri int

const (
	Reset Sakura = iota
	Mute
	Rose
	Love
	Gold
	Tree
	Iris
	Foam
	Pine
	Text
)
const (
	Unset Tri = iota
	False
	True
)

type Highlight struct {
	Color Sakura
	Bold  Tri
	Ital  Tri
}

type Chunk struct {
	Hl   *Highlight
	Text string
}

func (hl Highlight) Render() string {
	mods := []string{}
	switch hl.Color {
	case Mute:
		mods = append(mods, "2")
	case Text:
		mods = append(mods, "39")
	case Foam:
		mods = append(mods, []string{"36", "2"}...)
	case Love:
		mods = append(mods, "31")
	case Tree:
		mods = append(mods, "32")
	case Gold:
		mods = append(mods, "33")
	case Iris:
		mods = append(mods, "34")
	case Rose:
		mods = append(mods, "35")
	case Pine:
		mods = append(mods, "36")
	case Reset:
		mods = append(mods, "0")
	default:
		log.Fatalf("not handled: %d", hl.Color)
	}

	switch hl.Bold {
	case True:
		mods = append(mods, "1")
	case False:
		mods = append(mods, "22")
	}

	switch hl.Ital {
	case True:
		mods = append(mods, "3")
	case False:
		mods = append(mods, "23")
	}
	return fmt.Sprintf("\x1b[%sm", strings.Join(mods, ";"))
}

func (hl Highlight) Wrap(text string) string {
	return fmt.Sprintf("%s%s\x1b[0m", hl.Render(), text)
}

func (hl *Highlight) Merge(other Highlight) {
	if hl.Color != other.Color && other.Color != Reset {
		hl.Color = other.Color
	}
	if other.Bold != Unset {
		hl.Bold = other.Bold
	}
	if other.Ital != Unset {
		hl.Ital = other.Ital
	}
}

func fixupSyntax(out string) []string {
	lines := []string{}
	inBlock := false
	inComment := "\x00"

	tokens := []string{
		"const",
		"var",
		"type",
		"func",
		"package",
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		l := strings.TrimSpace(line)
		if l == "" {
			if inComment != "\x00" {
				l = strings.TrimSpace(lines[len(lines)-1])
				if strings.HasPrefix(l, "/*") {
					lines[len(lines)-1] = lines[len(lines)-1] + " */"
				} else {
					lines = append(lines, inComment+" */")
				}
				inComment = "\x00"
			}
			lines = append(lines, "")
			continue
		}
		words := strings.Split(l, " ")

		if !inBlock && inComment == "\x00" && slices.Index(tokens, words[0]) == -1 {
			inComment = strings.ReplaceAll(line[:strings.Index(line, l)], "    ", "\t")
			if len(lines) > 2 && strings.HasSuffix(lines[len(lines)-2], "*/") {
				st := lines[len(lines)-2]
				lines[len(lines)-2] = st[:len(st)-2]
				if strings.TrimSpace(lines[len(lines)-2]) == "" {
					lines = slices.Concat(lines[:len(lines)-2], lines[len(lines)-1:])
				}
				inComment = st[:strings.Index(st, "*")-1]
				lines[len(lines)-1] = inComment + " *"
			} else if inComment != "" {
				lines = append(lines, inComment+"/* "+l)
				continue
			} else {
				lines = append(lines, inComment+"/*")
			}
		}

		if inComment != "\x00" {
			indentSz := strings.Count(inComment, " ") + strings.Count(inComment, "\t")*4
			lines = append(lines, inComment+" * "+line[indentSz:])
			continue
		}

		if l[len(l)-1] == '{' || line[len(l)-1] == '(' {
			inBlock = true
		} else if l[0] == '}' || l[0] == ')' {
			inBlock = false
		} else if line[0] != '\t' && line[0] != ' ' && (len(lines) == 0 || lines[len(lines)-1] != "") {
			lines = append(lines, "")
		}
		lines = append(lines, line)
	}

	return lines
}

var highlights = map[string]Highlight{
	"@type":                  {Foam, Unset, Unset},
	"@type.definition":       {Foam, Unset, Unset},
	"@property":              {Foam, Unset, True},
	"@variable":              {Text, Unset, True},
	"@module":                {Text, Unset, Unset},
	"@variable.parameter":    {Iris, Unset, True},
	"@label":                 {Foam, Unset, Unset},
	"@constant":              {Gold, Unset, Unset},
	"@function.call":         {Rose, Unset, Unset},
	"@function.method.call":  {Iris, Unset, Unset},
	"@function":              {Rose, Unset, Unset},
	"@function.method":       {Rose, Unset, Unset},
	"@constructor":           {Foam, Unset, Unset},
	"@operator":              {Mute, Unset, Unset},
	"@keyword":               {Pine, Unset, Unset},
	"@keyword.type":          {Pine, Unset, Unset},
	"@keyword.function":      {Pine, Unset, Unset},
	"@keyword.return":        {Pine, Unset, Unset},
	"@keyword.coroutine":     {Pine, Unset, Unset},
	"@keyword.repeat":        {Pine, Unset, Unset},
	"@keyword.import":        {Pine, Unset, Unset},
	"@keyword.conditional":   {Pine, Unset, Unset},
	"@type.builtin":          {Foam, True, Unset},
	"@function.builtin":      {Rose, True, Unset},
	"@punctuation.delimiter": {Mute, Unset, Unset},
	"@punctuation.bracket":   {Mute, Unset, Unset},
	"@string":                {Gold, Unset, Unset},
	"@string.escape":         {Pine, Unset, Unset},
	"@number":                {Gold, Unset, Unset},
	"@number.float":          {Gold, Unset, Unset},
	"@boolean":               {Rose, Unset, Unset},
	"@constant.builtin":      {Gold, True, Unset},
	"@variable.member":       {Foam, Unset, Unset},
	"@spell":                 {Reset, Unset, Unset},
	"@comment.documentation": {Mute, Unset, True},
	"@comment":               {Mute, Unset, True},
	"@string.regexp":         {Iris, Unset, Unset},
}

const goQuery = `
; Forked from tree-sitter-go
; Copyright (c) 2014 Max Brunsfeld (The MIT License)
;
; Identifiers
(type_identifier) @type

(type_spec
  name: (type_identifier) @type.definition)

(field_identifier) @property

(identifier) @variable

(package_identifier) @module

(parameter_declaration
  (identifier) @variable.parameter)

(variadic_parameter_declaration
  (identifier) @variable.parameter)

(label_name) @label

(const_spec
  name: (identifier) @constant)

; Function calls
(call_expression
  function: (identifier) @function.call)

(call_expression
  function: (selector_expression
    field: (field_identifier) @function.method.call))

; Function definitions
(function_declaration
  name: (identifier) @function)

(method_declaration
  name: (field_identifier) @function.method)

(method_elem
  name: (field_identifier) @function.method)

; Constructors
((call_expression
  (identifier) @constructor)
  (#lua-match? @constructor "^[nN]ew.+$"))

((call_expression
  (identifier) @constructor)
  (#lua-match? @constructor "^[mM]ake.+$"))

; Operators
[
  "--"
  "-"
  "-="
  ":="
  "!"
  "!="
  "..."
  "*"
  "*"
  "*="
  "/"
  "/="
  "&"
  "&&"
  "&="
  "&^"
  "&^="
  "%"
  "%="
  "^"
  "^="
  "+"
  "++"
  "+="
  "<-"
  "<"
  "<<"
  "<<="
  "<="
  "="
  "=="
  ">"
  ">="
  ">>"
  ">>="
  "|"
  "|="
  "||"
  "~"
] @operator

; Keywords
[
  "break"
  "const"
  "continue"
  "default"
  "defer"
  "goto"
  "range"
  "select"
  "var"
  "fallthrough"
] @keyword

[
  "type"
  "struct"
  "interface"
] @keyword.type

"func" @keyword.function

"return" @keyword.return

"go" @keyword.coroutine

"for" @keyword.repeat

[
  "import"
  "package"
] @keyword.import

[
  "else"
  "case"
  "switch"
  "if"
] @keyword.conditional

; Builtin types
[
  "chan"
  "map"
] @type.builtin

((type_identifier) @type.builtin
  (#any-of? @type.builtin
    "any" "bool" "byte" "comparable" "complex128" "complex64" "error" "float32" "float64" "int"
    "int16" "int32" "int64" "int8" "rune" "string" "uint" "uint16" "uint32" "uint64" "uint8"
    "uintptr"))

; Builtin functions
((identifier) @function.builtin
  (#any-of? @function.builtin
    "append" "cap" "clear" "close" "complex" "copy" "delete" "imag" "len" "make" "max" "min" "new"
    "panic" "print" "println" "real" "recover"))

; Delimiters
"." @punctuation.delimiter

"," @punctuation.delimiter

":" @punctuation.delimiter

";" @punctuation.delimiter

"(" @punctuation.bracket

")" @punctuation.bracket

"{" @punctuation.bracket

"}" @punctuation.bracket

"[" @punctuation.bracket

"]" @punctuation.bracket

; Literals
(interpreted_string_literal) @string

(raw_string_literal) @string

(rune_literal) @string

(escape_sequence) @string.escape

(int_literal) @number

(float_literal) @number.float

(imaginary_literal) @number

[
  (true)
  (false)
] @boolean

[
  (nil)
  (iota)
] @constant.builtin

(keyed_element
  .
  (literal_element
    (identifier) @variable.member))

(field_declaration
  name: (field_identifier) @variable.member)

; Comments
(comment) @comment @spell

; Doc Comments
(source_file
  .
  (comment)+ @comment.documentation)

(source_file
  (comment)+ @comment.documentation
  .
  (const_declaration))

(source_file
  (comment)+ @comment.documentation
  .
  (function_declaration))

(source_file
  (comment)+ @comment.documentation
  .
  (type_declaration))

(source_file
  (comment)+ @comment.documentation
  .
  (var_declaration))

; Spell
((interpreted_string_literal) @spell
  (#not-has-parent? @spell import_spec))

; Regex
(call_expression
  (selector_expression) @_function
  (#any-of? @_function
    "regexp.Match" "regexp.MatchReader" "regexp.MatchString" "regexp.Compile" "regexp.CompilePOSIX"
    "regexp.MustCompile" "regexp.MustCompilePOSIX")
  (argument_list
    .
    [
      (raw_string_literal
        (raw_string_literal_content) @string.regexp)
      (interpreted_string_literal
        (interpreted_string_literal_content) @string.regexp)
    ]))
`
