package builder

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasFuncLiteral(t *testing.T) {
	t.Run("complex code", func(t *testing.T) {
		fset := token.NewFileSet()

		astFile, err := parser.ParseFile(fset, "", complexCode, parser.ParseComments)

		assert.Equal(t, nil, err)

		hasClosure, _ := hasFuncLiteral(astFile)

		// out := AstToBytes(astFile, fset)
		// t.Log(out)
		assert.Equal(t, true, hasClosure)
	})

	t.Run("no closure code", func(t *testing.T) {
		fset := token.NewFileSet()

		astFile, err := parser.ParseFile(fset, "", `
		package test2
		func namedFunction() int {
			return 3
		}
		func main() {
			LOOP:
			for x := 1; x <= namedFunction(); x = x^x {
				break LOOP
			}
		}
		`, parser.ParseComments)

		assert.Equal(t, nil, err)

		hasClosure, _ := hasFuncLiteral(astFile)

		// out := AstToBytes(astFile, fset)
		// t.Log(out)
		assert.Equal(t, false, hasClosure)
	})
}

// it will fall into infinite loop for unkonown reasons
func TestAddCounters(t *testing.T) {
	t.Run("simple code", func(t *testing.T) {
		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, "", code, parser.ParseComments)

		assert.Equal(t, nil, err)

		visitor := NewVisitorPtr(fset)
		ast.Walk(visitor, astFile)

		visitor.AddImportDecl(astFile)

		out := AstToBytes(astFile, fset)

		lines := strings.Fields(out.String())
		assert.Equal(t, lines[25], "__tidb_go_fuzz_dep.GetTraceTable().AddCount(0,")
	})

	t.Run("more cases", func(t *testing.T) {

		fset := token.NewFileSet()

		t.Log("parsing")

		astFile, err := parser.ParseFile(fset, "", complexCode, parser.ParseComments)

		assert.Equal(t, nil, err)

		visitor := NewVisitorPtr(fset)
		ast.Walk(visitor, astFile)

		visitor.AddImportDecl(astFile)

		_ := AstToBytes(astFile, fset)
		// need to check manually

	})
}

const complexCode = `
package test1

import (
	"fmt"
	"go/ast"
	"math"
)

var Function1 = func(f func(int) func(), args ...int) int {
	return len(args)
}

const Const1 = "$#$%%&^@#$@#$@#$" // ignore

func Function2() func() int {
	return func() int {
		return 1
	}
}

func Function3() {
	defer func() {
		fmt.Print("defer")
	}()

	if a := Function2()(); a > 0 {
		fmt.Print(1)
	}

	x := 0x99 & func() int {
		return Function2()()
	}()
// Loop:

	for j := -1; j < Function2()(); j = Function2()() & (x + 1) {
		switch b := Function1(func(int) func() { return func() {} }, 1, 3, 5, 6); -b {
		case Function2()():
		case -3:
			fmt.Print("case 1&2")
			// break Loop
		case -4:
			fmt.Print("case 3")
		case x:
			fmt.Print("case 4")
			fallthrough
		default:
			fmt.Print("default")
		}

		fmt.Print("do nothing")
	}
	var t1 ast.Node = new(ast.TypeAssertExpr)
	switch t1.(type) {
	case *ast.ArrayType:
	case *ast.BadDecl:
		fmt.Print("do nothing")
		return
	}

	var ch chan int
	select {
	case l := <-ch:
		fmt.Print(l)
		return
	case ch <- x:
	default:
		fmt.Print(1)
	}

	for ;; {
		if x := Function2()(); float64(x) < math.Abs(func(x float64) float64 { return x }(3.0)) {
			Function1(func(i int) func() { return func() {} }, x)
		} else if true {
			fmt.Print(x)
			break
		}
		return
	}
}

func main() {
	go func() {
		Function3()
	}()

	defer Function3()
}
`
