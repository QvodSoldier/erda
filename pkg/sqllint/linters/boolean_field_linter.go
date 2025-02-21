// Copyright (c) 2021 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package linters

import (
	"bytes"
	"strings"

	"github.com/pingcap/parser/ast"

	"github.com/erda-project/erda/pkg/sqllint/linterror"
	"github.com/erda-project/erda/pkg/sqllint/rules"
	"github.com/erda-project/erda/pkg/sqllint/script"
	"github.com/erda-project/erda/pkg/swagger/ddlconv"
)

type BooleanFieldLinter struct {
	baseLinter
}

func NewBooleanFieldLinter(script script.Script) rules.Rule {
	return &BooleanFieldLinter{baseLinter: newBaseLinter(script)}
}

func (l *BooleanFieldLinter) Enter(in ast.Node) (ast.Node, bool) {
	if l.text == "" || in.Text() != "" {
		l.text = in.Text()
	}

	col, ok := in.(*ast.ColumnDef)
	if !ok {
		return in, false
	}

	colName := ddlconv.ExtractColName(col)
	colType := ddlconv.ExtractColType(col)
	switch colType {
	case "bool", "boolean", "tinyint(1)", "bit":
		if !(strings.HasPrefix(colName, "is_") || strings.HasPrefix(colName, "has_")) {
			l.err = linterror.New(l.s, l.text, "boolean field should start with linking-verb, e.g. is_deleted, has_child",
				func(line []byte) bool {
					return bytes.Contains(line, []byte(colName))
				})
			return in, true
		}
	}

	if strings.HasPrefix(colName, "is_") || strings.HasPrefix(colName, "has_") {
		switch colType {
		case "bool", "boolean", "tinyint(1)", "bit":
			return in, true
		default:
			l.err = linterror.New(l.s, l.text, "boolean field type should be tinyint(1) or boolean",
				func(line []byte) bool {
					return bytes.Contains(line, []byte(colName))
				})
			return in, true
		}
	}

	return in, true
}

func (l *BooleanFieldLinter) Leave(in ast.Node) (ast.Node, bool) {
	return in, l.err == nil
}

func (l *BooleanFieldLinter) Error() error {
	return l.err
}
