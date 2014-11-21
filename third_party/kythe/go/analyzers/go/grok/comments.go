package grok

// This file defines a visitor on the Go AST that generates the Grok artifacts
// for documentation comments.  See http://go/coda for details.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"text/template" // for HTMLEscape
	"unicode"
	"unicode/utf8"

	"code.google.com/p/goprotobuf/proto"

	enumpb "third_party/kythe/proto/enums_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
)

// A commentWalker is an ast.Visitor that generates Coda comment artifacts.
type commentWalker struct{ *context }

// Visit implements ast.Visitor.
func (w commentWalker) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.File:
		w.visitFile(t)
	case *ast.FuncDecl:
		w.visitFuncDecl(t)
	case *ast.TypeSpec:
		w.visitTypeSpec(t)
	case *ast.ValueSpec:
		w.visitValueSpec(t)
	case *ast.Field:
		w.visitField(t)
	case *ast.GenDecl:
		w.visitGenDecl(t)
	}
	return w
}

// visitFile creates comment artifacts for a package (file) comment.
func (w commentWalker) visitFile(file *ast.File) {
	if ticket := w.addComment(w.fileTicket(file), file.Doc); ticket != "" {
		w.addEdges(ticket, w.packageTicket(w.Package),
			enumpb.EdgeEnum_DOCUMENTS, enumpb.EdgeEnum_DOCUMENTED_WITH, noPos)
	}
}

// visitFuncDecl creates comment artifacts for a function declaration.
func (w commentWalker) visitFuncDecl(decl *ast.FuncDecl) {
	if obj := w.TypeInfo.Defs[decl.Name]; obj != nil {
		w.addComment(w.objectTicket(obj), decl.Doc)
	}
}

// visitTypeSpec creates comment artifacts for a type specification.
func (w commentWalker) visitTypeSpec(spec *ast.TypeSpec) {
	if obj := w.TypeInfo.Defs[spec.Name]; obj != nil {
		w.addComment(w.objectTicket(obj), spec.Doc)
	}
}

// visitValueSpec creates comment artifacts for a value specification.
func (w commentWalker) visitValueSpec(spec *ast.ValueSpec) {
	// Go promises at least 1 name.
	if obj := w.TypeInfo.Defs[spec.Names[0]]; obj != nil {
		w.addComment(w.objectTicket(obj), spec.Doc)
	}
}

// visitField creates comment artifacts for a field specification.
func (w commentWalker) visitField(spec *ast.Field) {
	if len(spec.Names) > 0 {
		if obj := w.TypeInfo.Defs[spec.Names[0]]; obj != nil {
			w.addComment(w.objectTicket(obj), spec.Doc)
		}
	}
}

// visitGenDecl checks that documentation for the declared types or values are
// set to something sensible.
func (w commentWalker) visitGenDecl(decl *ast.GenDecl) {
	if decl.Doc == nil {
		return
	}
	for _, spec := range decl.Specs {
		switch t := spec.(type) {
		case *ast.TypeSpec:
			if t.Doc == nil {
				t.Doc = decl.Doc
			}
		case *ast.ValueSpec:
			if t.Doc == nil {
				t.Doc = decl.Doc
			}
		}
	}
}

// addComment creates and writes a comment node for the specified node, using
// the given comment group as the text.  If either node or cg is nil, this has
// no effect.  If the comment is created, edges are written to join it to its
// affected node.  Returns the ticket of the comment node, or "".
func (w commentWalker) addComment(ticket string, cg *ast.CommentGroup) string {
	if ticket == "" || cg == nil {
		return ""
	}
	text := commentText(cg)
	if len(text) == 0 {
		return ""
	}
	body := commentBody(text)
	pos, end := commentBounds(cg)
	comment := w.makeCommentNode(ticket, text, pos, end)
	w.writeNode(comment)
	w.addEdges(comment.GetTicket(), ticket,
		enumpb.EdgeEnum_DOCUMENTS, enumpb.EdgeEnum_DOCUMENTED_WITH, noPos)
	w.addCommentStructure(comment, body)
	return comment.GetTicket()
}

// commentText extracts the full text of all the comments in the given comment
// group, including any leading markers.
func commentText(cg *ast.CommentGroup) []string {
	if cg == nil {
		return nil
	}

	var text []string
	for _, c := range cg.List {
		text = append(text, c.Text)
	}
	return text
}

// commentBody extracts the un-marked body of a comment from its parts.
func commentBody(parts []string) []string {
	var body []string
	for _, part := range parts {
		if strings.HasPrefix(part, "/*") {
			part = part[2 : len(part)-2] // Trim /* ... */ markers.
		} else if strings.HasPrefix(part, "//") {
			part = part[2:] // Trim leading // marker.
		}
		body = append(body, part)
	}
	return body
}

// commentBounds returns the starting and ending position of the comment group,
// which is from the start of the first comment to the end of the last.  If the
// group is nil or empty, the bounds are token.NoPos.
func commentBounds(cg *ast.CommentGroup) (pos, end token.Pos) {
	if cg != nil && len(cg.List) > 0 {
		pos = cg.List[0].Slash
		last := cg.List[len(cg.List)-1]
		end = last.Slash + token.Pos(len(last.Text))
	}
	return
}

// makeCommentNode returns a COMMENT node for the given comment text.  If pos
// and end are valid, they are used to set the location of the comment.  The
// ticket is the ticket of the commented node.
func (w commentWalker) makeCommentNode(ticket string, text []string, pos, end token.Pos) *kythepb.Node {
	source := []byte(strings.Join(text, "\n"))
	return &kythepb.Node{
		Kind:     enumpb.NodeEnum_COMMENT.Enum(),
		Ticket:   proto.String(ticket + "⇐comment"),
		Location: w.makePosLoc(pos, end),
		Language: enumpb.Language_GO.Enum(),
		Content: &kythepb.NodeContent{
			SourceText: &kythepb.SourceText{
				EncodedText: source,
				Encoding:    proto.String("UTF-8"),
			},
		},
	}
}

// addCommentStructure generates the DOCUMENTATION and TEXT nodes for the given
// comment node, given the un-marked body of the comment text.
func (w commentWalker) addCommentStructure(comment *kythepb.Node, body []string) {
	// The DOCUMENTATION node encapsulates the tree structure of the comment.
	// For Go, there is no structure, so it has only one child.
	doc := &kythepb.Node{
		Kind:     enumpb.NodeEnum_DOCUMENTATION.Enum(),
		Ticket:   proto.String(comment.GetTicket() + ".doc"),
		Location: comment.Location,
		Language: enumpb.Language_GO.Enum(),
	}

	w.toHTML(strings.Join(body, "\n"), doc)

	w.writeNode(doc)
	w.addEdges(comment.GetTicket(), doc.GetTicket(),
		enumpb.EdgeEnum_TREE_PARENT, enumpb.EdgeEnum_TREE_CHILD, noPos)
}

// toHTML converts comment text to formatted HTML.
// The comment is expected not to have leading, trailing blank lines
// nor to have trailing spaces at the end of lines.
// The comment markers have already been removed.
//
// Turn each run of multiple \n into </p><p>.
// Turn each run of indented lines into a <pre> block without indent.
// Enclose headings with header tags.
func (w commentWalker) toHTML(text string, parent *kythepb.Node) {
	for i, b := range blocks(text) {
		switch b.op {
		case opPara:
			para := htmlTag("p", fmt.Sprintf("%s.tag@%d", parent.GetTicket(), i), true, parent.Location)
			w.addNodeToCommentTree(para, parent.GetTicket(), i)
			// The TEXT node for paragraph.
			w.addNodeToCommentTree(textNode(commentEscape(strings.Join(b.lines, ""), true), para.GetTicket()+".text", parent.Location),
				para.GetTicket(), 0)
			// Writing the end tag for <p>
			w.addNodeToCommentTree(htmlTag("p", fmt.Sprintf("%s.endtag@%d", para.GetTicket(), i), false, parent.Location),
				para.GetTicket(), 1)
		case opHead:
			head := htmlTag("h3", fmt.Sprintf("%s.tag@%d", parent.GetTicket(), i), true, parent.Location)
			w.addNodeToCommentTree(head, parent.GetTicket(), i)
			// Writing the content of heading
			w.addNodeToCommentTree(textNode(commentEscape(strings.Join(b.lines, ""), true), head.GetTicket()+".text", parent.Location),
				head.GetTicket(), 0)
			// Writing the end tag for <h3>
			w.addNodeToCommentTree(htmlTag("h3", fmt.Sprintf("%s.endtag@%d", head.GetTicket(), i), false, parent.Location),
				head.GetTicket(), 1)
		case opPre:
			pre := htmlTag("pre", fmt.Sprintf("%s.tag@%d", parent.GetTicket(), i), true, parent.Location)
			w.addNodeToCommentTree(pre, parent.GetTicket(), i)
			// The TEXT node for pre
			w.addNodeToCommentTree(textNode(commentEscape(strings.Join(b.lines, ""), true), pre.GetTicket()+".text", parent.Location),
				pre.GetTicket(), 0)
			// Writing the end tag for <pre>
			w.addNodeToCommentTree(htmlTag("pre", fmt.Sprintf("%s.endtag@%d", pre.GetTicket(), i), false, parent.Location),
				pre.GetTicket(), 1)
		}
	}
}

func (w commentWalker) addNodeToCommentTree(node *kythepb.Node, parent string, pos int) {
	w.writeNode(node)
	w.addEdges(parent, node.GetTicket(), enumpb.EdgeEnum_TREE_PARENT, enumpb.EdgeEnum_TREE_CHILD, pos)
}

func htmlTag(name string, ticket string, open bool, location *kythepb.Location) *kythepb.Node {
	tag := &kythepb.Node{
		Kind:       enumpb.NodeEnum_MARKUP_TAG.Enum(),
		Ticket:     proto.String(ticket),
		Identifier: proto.String(name),
		Language:   enumpb.Language_GO.Enum(),
		Location:   location,
		Modifiers: &kythepb.Modifiers{
			OpenDelimiter:  proto.Bool(open),
			CloseDelimiter: proto.Bool(!open),
		},
	}
	return tag
}

func textNode(source string, ticket string, location *kythepb.Location) *kythepb.Node {
	text := &kythepb.Node{
		Kind:     enumpb.NodeEnum_TEXT.Enum(),
		Ticket:   proto.String(ticket),
		Language: enumpb.Language_GO.Enum(),
		Location: location,
		Content: &kythepb.NodeContent{
			SourceText: &kythepb.SourceText{
				EncodedText: []byte(source),
				Encoding:    proto.String("UTF-8"),
			},
		},
	}
	return text
}

// All the following functions are copied from go/doc/comment.go.

var (
	ldquo = []byte("&ldquo;")
	rdquo = []byte("&rdquo;")
)

// Escape comment text for HTML. If nice is set,
// also turn `` into &ldquo; and '' into &rdquo;.
func commentEscape(text string, nice bool) string {
	var buf bytes.Buffer
	last := 0
	if nice {
		for i := 0; i < len(text)-1; i++ {
			ch := text[i]
			if ch == text[i+1] && (ch == '`' || ch == '\'') {
				template.HTMLEscape(&buf, []byte(text[last:i]))
				last = i + 2
				switch ch {
				case '`':
					buf.Write(ldquo)
				case '\'':
					buf.Write(rdquo)
				}
				i++ // loop will add one more
			}
		}
	}
	template.HTMLEscape(&buf, []byte(text[last:]))
	return buf.String()
}

func indentLen(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func isBlank(s string) bool {
	return len(s) == 0 || (len(s) == 1 && s[0] == '\n')
}

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[0:i]
}

func unindent(block []string) {
	if len(block) == 0 {
		return
	}

	// compute maximum common white prefix
	prefix := block[0][0:indentLen(block[0])]
	for _, line := range block {
		if !isBlank(line) {
			prefix = commonPrefix(prefix, line[0:indentLen(line)])
		}
	}
	n := len(prefix)

	// remove
	for i, line := range block {
		if !isBlank(line) {
			block[i] = line[n:]
		}
	}
}

// heading returns the trimmed line if it passes as a section heading;
// otherwise it returns the empty string.
func heading(line string) string {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return ""
	}

	// a heading must start with an uppercase letter
	r, _ := utf8.DecodeRuneInString(line)
	if !unicode.IsLetter(r) || !unicode.IsUpper(r) {
		return ""
	}

	// it must end in a letter or digit:
	r, _ = utf8.DecodeLastRuneInString(line)
	if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		return ""
	}

	// exclude lines with illegal characters
	if strings.IndexAny(line, ",.;:!?+*/=()[]{}_^°&§~%#@<\">\\") >= 0 {
		return ""
	}

	// allow "'" for possessive "'s" only
	for b := line; ; {
		i := strings.IndexRune(b, '\'')
		if i < 0 {
			break
		}
		if i+1 >= len(b) || b[i+1] != 's' || (i+2 < len(b) && b[i+2] != ' ') {
			return "" // not followed by "s "
		}
		b = b[i+2:]
	}

	return line
}

type op int

const (
	opPara op = iota
	opHead
	opPre
)

type block struct {
	op    op
	lines []string
}

var nonAlphaNumRx = regexp.MustCompile(`[^a-zA-Z0-9]`)

func blocks(text string) []block {
	var (
		out  []block
		para []string

		lastWasBlank   = false
		lastWasHeading = false
	)

	close := func() {
		if para != nil {
			out = append(out, block{opPara, para})
			para = nil
		}
	}

	lines := strings.SplitAfter(text, "\n")
	unindent(lines)
	for i := 0; i < len(lines); {
		line := lines[i]
		if isBlank(line) {
			// close paragraph
			close()
			i++
			lastWasBlank = true
			continue
		}
		if indentLen(line) > 0 {
			// close paragraph
			close()

			// count indented or blank lines
			j := i + 1
			for j < len(lines) && (isBlank(lines[j]) || indentLen(lines[j]) > 0) {
				j++
			}
			// but not trailing blank lines
			for j > i && isBlank(lines[j-1]) {
				j--
			}
			pre := lines[i:j]
			i = j

			unindent(pre)

			// put those lines in a pre block
			out = append(out, block{opPre, pre})
			lastWasHeading = false
			continue
		}

		if lastWasBlank && !lastWasHeading && i+2 < len(lines) &&
			isBlank(lines[i+1]) && !isBlank(lines[i+2]) && indentLen(lines[i+2]) == 0 {
			// current line is non-blank, surrounded by blank lines
			// and the next non-blank line is not indented: this
			// might be a heading.
			if head := heading(line); head != "" {
				close()
				out = append(out, block{opHead, []string{head}})
				i += 2
				lastWasHeading = true
				continue
			}
		}

		// open paragraph
		lastWasBlank = false
		lastWasHeading = false
		para = append(para, lines[i])
		i++
	}
	close()

	return out
}
