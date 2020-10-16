package ssaUtils

import (
	"errors"
	"fmt"
	"github.com/amit-davidson/Chronos/domain"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"os"
	"sort"
	"strings"
)

var GlobalProgram *ssa.Program
var GlobalPackageName string

var typesCache = make(map[*types.Interface][]*ssa.Function, 0)

func LoadPackage(path string) (*ssa.Program, *ssa.Package, error) {
	conf1 := packages.Config{
		Mode: packages.LoadAllSyntax,
	}
	loadQuery := fmt.Sprintf("file=%s", path)
	pkgs, err := packages.Load(&conf1, loadQuery)
	if err != nil {
		return nil, nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil, errors.New(fmt.Sprintf("Cannot load the given file: %s", path))
	}
	ssaProg, ssaPkgs := ssautil.AllPackages(pkgs, 0)
	ssaProg.Build()
	ssaPkg := ssaPkgs[0]
	return ssaProg, ssaPkg, nil
}
func SetGlobals(prog *ssa.Program, pkg *ssa.Package) error {
	var retError error
	GlobalProgram = prog
	GlobalPackageName, retError = GetTopLevelPackageName(pkg)
	if retError != nil {
		return retError
	}
	return nil
}

func GetTopLevelPackageName(pkg *ssa.Package) (string, error){
	pkgName := pkg.Pkg.Path()
	r := strings.SplitAfterN(pkgName, string(os.PathSeparator), 4)
	if len(r) < 3 {
		return "", errors.New("package should be provided in the following format:{host}/{organization}/{package}")
	}
	topLevelPackage := r[0] + r[1] + r[2]
	return topLevelPackage, nil
}

func GetMethodImplementations(recv types.Type, method *types.Func) []*ssa.Function {
	methodImplementations := make([]*ssa.Function, 0)
	recvInterface := recv.(*types.Interface)

	if methodImplementations, ok := typesCache[recvInterface]; ok {
		return methodImplementations
	}

	implementors := make([]types.Type, 0)
	for _, typ := range GlobalProgram.RuntimeTypes() {
		if types.Implements(typ, recvInterface) {
			implementors = append(implementors, typ)
		}
	}
	for _, implementor := range implementors {
		setMethods := GlobalProgram.MethodSets.MethodSet(implementor)
		method := setMethods.Lookup(method.Pkg(), method.Name())
		methodImpl := GlobalProgram.MethodValue(method)
		if methodImpl.Synthetic == "" {
			methodImplementations = append(methodImplementations, methodImpl)
		}
	}

	// Sort by pos to enter previous implementations first. This make the search deterministic and easier for debugging
	sortedImplementations := sortMethodImplementations(methodImplementations)
	typesCache[recvInterface] = sortedImplementations
	return sortedImplementations
}

func sortMethodImplementations(methodImplementations []*ssa.Function) []*ssa.Function {
	posSlice := make([]int, 0)
	sortedImplementations := make([]*ssa.Function, 0)
	implMap := make(map[int]*ssa.Function, 0)
	for _, methodImplementation := range methodImplementations {
		pos := methodImplementation.Pos()
		implMap[int(pos)] = methodImplementation
		posSlice = append(posSlice, int(pos))
	}
	sort.Ints(posSlice)
	for _, pos := range posSlice {
		sortedImplementations = append(sortedImplementations, implMap[pos])
	}
	return sortedImplementations
}

func GetStackTrace(prog *ssa.Program, ga *domain.GuardedAccess) string {
	stack := ""
	for _, pos := range ga.Stacktrace {
		calculatedPos := prog.Fset.Position(token.Pos(pos))
		stack += calculatedPos.String()
		stack += " ->\n"
	}
	return stack
}
