package archcheck

import (
	"go/ast"
	"go/token"
	"strings"
	"time"
)

type reviewMark struct {
	pos         token.Pos
	line        int
	text        string
	date        time.Time
	valid, used bool
}

func reviewMarks(fset *token.FileSet, file *ast.File) []*reviewMark {
	var marks []*reviewMark
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !strings.HasPrefix(comment.Text, "//gomad:dynamic") {
				continue
			}
			mark := &reviewMark{pos: comment.Pos(), line: fset.Position(comment.Pos()).Line, text: comment.Text}
			fields := strings.Fields(strings.TrimPrefix(comment.Text, "//gomad:dynamic"))
			if len(fields) >= 3 && fields[0] == "reviewed" {
				date, err := time.Parse(time.DateOnly, fields[1])
				mark.date, mark.valid = date, err == nil
			}
			marks = append(marks, mark)
		}
	}
	return marks
}

func useMark(fset *token.FileSet, pos token.Pos, marks []*reviewMark) bool {
	line := fset.Position(pos).Line
	for _, mark := range marks {
		if mark.valid && (mark.line == line || mark.line == line-1) {
			mark.used = true
			return true
		}
	}
	return false
}
