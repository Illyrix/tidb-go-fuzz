package builder

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/Illyrix/tidb-go-fuzz/dep/types"
)

// `import github.com/Illyrix/tidb-go-fuzz-dep as "tidb_go_fuzz_dep"``
const FUZZ_DEP_IMPORT_AS = "__tidb_go_fuzz_dep"

// then use `__tidb_go_fuzz_dep.GetTraceTable()` to fetch the singleton
// TraceTable instance

func makeCountNode(src, dst types.BlockIdType) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: FUZZ_DEP_IMPORT_AS,
						},
						Sel: &ast.Ident{
							Name: "GetTraceTable",
						},
					},
				},
				Sel: &ast.Ident{
					Name: "AddCount",
				},
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(uint64(src), 10),
				},
				&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(uint64(dst), 10),
				},
			},
		},
	}
}

// inject `import ".../tidb-go-fuzz/dep" as ...` into where Counter appears
func addImportDecl() {}

// inject calling `tidb_go_fuzz.Listen()` on startup
func startListen() {

}
