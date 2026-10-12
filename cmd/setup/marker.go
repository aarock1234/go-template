package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// markerKind classifies a line of a marked file.
type markerKind int

const (
	markerNone   markerKind = iota // an ordinary line
	markerOpen                     // a full-line "[tag]" comment opening a block
	markerClose                    // a full-line "[/tag]" comment closing a block
	markerInline                   // code followed by a comment of "[tag]" tokens
)

var (
	// blockMarker matches a trimmed line that holds only a block marker
	// comment, such as "# [postgres]" or "// [/server]".
	blockMarker = regexp.MustCompile(`^(?:#|//)\s*\[(/?)([a-z0-9-]+)\]$`)

	// inlineMarker matches code followed by a trailing comment made only of
	// tags, such as `"net/http" // [server]` or "PORT=8080 # [a] [b]". The
	// first group is the code and the second holds the tags.
	inlineMarker = regexp.MustCompile(`^(.*?\S)\s+(?:#|//)((?:\s*\[[a-z0-9-]+\])+)\s*$`)

	// tagToken matches a single "[tag]" in an inline marker, capturing the
	// tag name.
	tagToken = regexp.MustCompile(`\[([a-z0-9-]+)\]`)
)

// markedLine is one line of a marked file.
type markedLine struct {
	text   string // the line as written, without any inline marker comment
	marker markerKind
	tags   []tag // the tag of a block marker, or every tag of an inline marker
}

// hasAny reports whether l carries at least one of tags.
func (l markedLine) hasAny(tags []tag) bool {
	for _, t := range l.tags {
		if slices.Contains(tags, t) {
			return true
		}
	}

	return false
}

// parseMarkers splits data, which must use "\n" line endings, into lines and
// classifies each one. It returns an error if a block marker is unbalanced or
// blocks overlap without nesting.
func parseMarkers(data string) ([]markedLine, error) {
	var (
		lines []markedLine
		open  []int // indexes of the open block markers, innermost last
	)

	for i, raw := range strings.Split(data, "\n") {
		l := parseLine(raw)

		switch l.marker {
		case markerOpen:
			open = append(open, i)
		case markerClose:
			if len(open) == 0 || lines[open[len(open)-1]].tags[0] != l.tags[0] {
				return nil, fmt.Errorf("line %d: [/%s] does not close the innermost open block", i+1, l.tags[0])
			}

			open = open[:len(open)-1]
		}

		lines = append(lines, l)
	}

	if len(open) > 0 {
		start := open[len(open)-1]

		return nil, fmt.Errorf("line %d: [%s] is never closed", start+1, lines[start].tags[0])
	}

	return lines, nil
}

// parseLine classifies a single line without its trailing newline.
func parseLine(raw string) markedLine {
	if match := blockMarker.FindStringSubmatch(strings.TrimSpace(raw)); match != nil {
		marker := markerOpen
		if match[1] == "/" {
			marker = markerClose
		}

		return markedLine{
			text:   raw,
			marker: marker,
			tags:   []tag{tag(match[2])},
		}
	}

	match := inlineMarker.FindStringSubmatch(raw)
	if match == nil {
		return markedLine{text: raw}
	}

	var tags []tag
	for _, token := range tagToken.FindAllStringSubmatch(match[2], -1) {
		tags = append(tags, tag(token[1]))
	}

	return markedLine{
		text:   match[1],
		marker: markerInline,
		tags:   tags,
	}
}

// render joins lines with "\n" into file contents without the blocks and
// inline lines tagged with any of removed, and without any remaining markers.
// A blank line that would directly follow another blank line, or start the
// file, only because lines between them were dropped is dropped as well.
func render(lines []markedLine, removed []tag) string {
	var (
		out     []string
		depth   int  // nesting depth inside a removed block, 0 outside
		dropped bool // whether a line was dropped since the last kept line
	)

	for _, l := range lines {
		switch {
		case depth > 0:
			// Track nested blocks to find the end of the removed one.
			switch l.marker {
			case markerOpen:
				depth++
			case markerClose:
				depth--
			}

			dropped = true
		case l.marker == markerOpen || l.marker == markerClose:
			// Marker lines never survive; a removed block's contents go too.
			if l.marker == markerOpen && l.hasAny(removed) {
				depth = 1
			}

			dropped = true
		case l.marker == markerInline && l.hasAny(removed):
			dropped = true
		case dropped && isBlank(l.text) && endsBlank(out):
			// Skip a blank line that would double one left by dropped lines.
		default:
			out = append(out, l.text)
			dropped = false
		}
	}

	return strings.Join(out, "\n")
}

// isBlank reports whether s contains only whitespace.
func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// endsBlank reports whether lines is empty or ends with a blank line.
func endsBlank(lines []string) bool {
	return len(lines) == 0 || isBlank(lines[len(lines)-1])
}
