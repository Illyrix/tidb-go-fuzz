package builder

import (
	"go/ast"
	"go/token"
	"math/rand"
	"time"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
)

type Visitor struct {
	// blockIds []types.BlockIdType // current block id stack
	// blocks   []*types.Block      // all blocks in this file (unordered)

	// the outer block is 0x0000
	parentBlockId types.BlockIdType

	FSet *token.FileSet
}

func init() {
	rand.Seed(time.Now().Unix())
}

// for recursive visit
func (v *Visitor) Clone() *Visitor {
	if v == nil {
		return nil
	}
	return &Visitor{
		parentBlockId: v.parentBlockId,
		FSet:          v.FSet,
	}
}

func NewVisitorPtr(fset *token.FileSet) *Visitor {
	return &Visitor{
		FSet:          fset,
		parentBlockId: 0,
	}
}

func (v *Visitor) Visit(n ast.Node) ast.Visitor {
	// fmt.Printf("%T\n", n)
	switch t := n.(type) {
	case *ast.GenDecl:
		if t.Tok != token.VAR {
			return nil
		}
	case *ast.FuncDecl:
		if t.Name.String() == "init" {
			// init function only always run once
			return nil
		}
	case *ast.SwitchStmt:
		// Same as TypeSwitchStmt
		// Don't annotate an empty switch - creates a syntax error.
		if t.Body == nil || len(t.Body.List) == 0 {
			return nil
		}
		hasDefault := false
		for _, s := range t.Body.List {
			if cas, ok := s.(*ast.CaseClause); ok && cas.List == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			// switch { case: ... }
			// ==>
			// switch { case: ... default: { /*empty code block*/ } }
			t.Body.List = append(t.Body.List, &ast.CaseClause{})
		}
		// see https://github.com/dvyukov/go-fuzz/blob/ea4a322d67f6e874238a8a7ab28e95a6d6675190/go-fuzz-build/cover.go#L80
		// for why go-fuzz needs replacement
	case *ast.TypeSwitchStmt:
		// Don't annotate an empty switch - creates a syntax error.
		if t.Body == nil || len(t.Body.List) == 0 {
			return nil
		}
		hasDefault := false
		for _, s := range t.Body.List {
			if cas, ok := s.(*ast.CaseClause); ok && cas.List == nil {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			// switch { case: ... }
			// ==>
			// switch { case: ... default: { /*empty code block*/ } }
			t.Body.List = append(t.Body.List, &ast.CaseClause{})
		}
		// see https://github.com/dvyukov/go-fuzz/blob/ea4a322d67f6e874238a8a7ab28e95a6d6675190/go-fuzz-build/cover.go#L80
		// for why go-fuzz needs replacement
		// may use `case interface{}:` is also effective?
	case *ast.IfStmt:
		if t.Init != nil {
			ast.Walk(v, t.Init)
		}
		if t.Cond != nil {
			ast.Walk(v, t.Cond)
		}
		ast.Walk(v, t.Body)
		if t.Else == nil {
			return nil
		}
		// if __COND_A__ { __BODY_A__ } else if __COND_B__ { __BODY_B__ }
		// ==>
		// if __COND_A__ { __BODY_A__ } else { if __COND_B__ { __BODY_B__ } }
		if e, ok := t.Else.(*ast.IfStmt); ok {
			t.Else = &ast.BlockStmt{
				Lbrace: t.Body.End(),
				List:   []ast.Stmt{e},
				Rbrace: e.End(),
			}
		}
		ast.Walk(v, t.Else)
		return nil
	case *ast.BlockStmt:
		if len(t.List) > 0 {
			switch t.List[0].(type) {
			case *ast.CaseClause: // switch
				for _, n := range t.List {
					clause := n.(*ast.CaseClause)
					_, clause.Body = v.addCounters(clause.Pos(), clause.End(), clause.Body, false)
				}
				return v
			case *ast.CommClause: // select
				for _, n := range t.List {
					clause := n.(*ast.CommClause)
					_, clause.Body = v.addCounters(clause.Pos(), clause.End(), clause.Body, false)
				}
				return v
			}
		}
		var blockId types.BlockIdType
		blockId, t.List = v.addCounters(t.Lbrace, t.Rbrace+1, t.List, true) // +1 to step past closing brace.
		cloned := v.Clone()
		cloned.parentBlockId = blockId
		return cloned
	case *ast.BinaryExpr:
		if t.Op == token.LAND || t.Op == token.LOR {
			// x || y ==> x || (func() bool {return y})()
			// see https://github.com/dvyukov/go-fuzz/blob/ea4a322d67f6e874238a8a7ab28e95a6d6675190/go-fuzz-build/cover.go#L607
			t.Y = &ast.CallExpr{
				Fun: &ast.FuncLit{
					Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}},
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{t.Y}}}},
				},
			}
		}
	}
	return v
}

func (v *Visitor) addCounters(pos, blockEnd token.Pos, stmts []ast.Stmt, extendToClosingBrace bool) (types.BlockIdType, []ast.Stmt) {
	// divide this block into several blocks by control flow statements. e.g.
	// { ... if 1>0 { ... } ... }
	// ==>
	// { ... BLOCK1 } if 1>0 { ... BLOCK2 } { ... BLOCK3 }

	if len(stmts) == 0 {
		bId := genBlockId()
		return bId, []ast.Stmt{v.newCounter(pos, blockEnd, v.parentBlockId, bId)}
	}

	list := make([]ast.Stmt, 0)
	lastBId := v.parentBlockId
	for {
		// find the first control flow statement, and it will be
		// the last stmt of current block
		last := 0
		end := blockEnd
		for last = 0; last < len(stmts); last++ {
			end = v.statementBoundary(stmts[last])
			if v.endsBasicSourceBlock(stmts[last]) {
				extendToClosingBrace = false // Block is broken up now.
				last++
				break
			}
		}
		if extendToClosingBrace {
			end = blockEnd
		}
		if pos != end { // Can have no source to cover if e.g. blocks abut.
			bId := genBlockId()
			list = append(list, v.newCounter(pos, end, lastBId, bId))
			lastBId = bId
		}
		list = append(list, stmts[0:last]...)
		stmts = stmts[last:]
		if len(stmts) == 0 {
			break
		}
		pos = stmts[0].Pos()
	}

	return lastBId, list
}

// Warn: its implementation relates to defination of BlockIdType
func genBlockId() types.BlockIdType {
	return types.BlockIdType(rand.Uint32() >> 16)
}

func (v *Visitor) newCounter(pos, end token.Pos, src, dst types.BlockIdType) ast.Stmt {
	// do not log where this block attaches
	return makeCountNode(src, dst)
}

func (v *Visitor) statementBoundary(s ast.Stmt) token.Pos {
	switch s := s.(type) {
	case *ast.BlockStmt:
		// Treat blocks like basic blocks to avoid overlapping counters.
		return s.Lbrace
	case *ast.IfStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Cond)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.ForStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Cond)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Post)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.LabeledStmt:
		return v.statementBoundary(s.Stmt)
	case *ast.RangeStmt:
		found, pos := hasFuncLiteral(s.X)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.SwitchStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		found, pos = hasFuncLiteral(s.Tag)
		if found {
			return pos
		}
		return s.Body.Lbrace
	case *ast.SelectStmt:
		return s.Body.Lbrace
	case *ast.TypeSwitchStmt:
		found, pos := hasFuncLiteral(s.Init)
		if found {
			return pos
		}
		return s.Body.Lbrace
	}
	found, pos := hasFuncLiteral(s)
	if found {
		return pos
	}
	return s.End()
}

type funcLitFinder token.Pos

func (f *funcLitFinder) Visit(node ast.Node) (w ast.Visitor) {
	if f.found() {
		return nil // Prune search.
	}
	switch n := node.(type) {
	case *ast.FuncLit:
		*f = funcLitFinder(n.Body.Lbrace)
		return nil // Prune search.
	}
	return f
}

func (f *funcLitFinder) found() bool {
	return token.Pos(*f) != token.NoPos
}

// find closure function
func hasFuncLiteral(n ast.Node) (bool, token.Pos) {
	if n == nil {
		return false, 0
	}
	var literal funcLitFinder
	ast.Walk(&literal, n)
	return literal.found(), token.Pos(literal)
}

func (v *Visitor) endsBasicSourceBlock(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.BlockStmt:
		// Treat blocks like basic blocks to avoid overlapping counters.
		return true
	case *ast.BranchStmt:
		return true
	case *ast.ForStmt:
		return true
	case *ast.IfStmt:
		return true
	case *ast.LabeledStmt:
		return v.endsBasicSourceBlock(s.Stmt)
	case *ast.RangeStmt:
		return true
	case *ast.SwitchStmt:
		return true
	case *ast.SelectStmt:
		return true
	case *ast.TypeSwitchStmt:
		return true
	case *ast.ExprStmt:
		// Calls to panic change the flow.
		// We really should verify that "panic" is the predefined function,
		// but without type checking we can't and the likelihood of it being
		// an actual problem is vanishingly small.
		if call, ok := s.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" && len(call.Args) == 1 {
				return true
			}
		}
	}
	found, _ := hasFuncLiteral(s)
	return found
}

// inject `import ".../tidb-go-fuzz/dep" as ...` into where Counter appears
func (v *Visitor) AddImportDecl(aFile *ast.File) {
	hasImports := false
	for _, decl := range aFile.Decls {
		if gDecl, ok := decl.(*ast.GenDecl); ok {
			if gDecl.Tok == token.IMPORT {
				hasImports = true
				gDecl.Specs = append(gDecl.Specs, &ast.ImportSpec{
					Path: &ast.BasicLit{Kind: token.STRING,
						Value: "\"" + FUZZ_DEP_IMPORT_NAME + "\""},
					Name: &ast.Ident{
						Name: FUZZ_DEP_IMPORT_AS,
					},
				})
				break
			}
		}
	}

	if !hasImports {
		newDecl := make([]ast.Decl, 0)
		newDecl = append(newDecl, &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{&ast.ImportSpec{
				Path: &ast.BasicLit{Kind: token.STRING,
					Value: "\"" + FUZZ_DEP_IMPORT_NAME + "\""},
				Name: &ast.Ident{
					Name: FUZZ_DEP_IMPORT_AS,
				},
			}},
		})
		newDecl = append(newDecl, aFile.Decls...)
		aFile.Decls = newDecl
	}
}
