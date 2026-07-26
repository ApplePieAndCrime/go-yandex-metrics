// Command reset generates Reset methods for structures marked with
// "// generate:reset".
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	directive     = "// generate:reset"
	generatedFile = "reset.gen.go"
)

type resetStruct struct {
	name       string
	receiver   string
	structType *ast.StructType
}

type packageToGenerate struct {
	name    string
	dir     string
	types   *types.Package
	info    *types.Info
	structs []resetStruct
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	overlay, err := generatedFilesOverlay(root)
	if err != nil {
		return err
	}

	cfg := &packages.Config{
		Dir:     root,
		Overlay: overlay,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Fset: token.NewFileSet(),
	}

	loaded, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}
	if n := packages.PrintErrors(loaded); n != 0 {
		return fmt.Errorf("cannot generate reset methods: found %d package errors", n)
	}

	packagesToGenerate, generatedTypes, err := findResetStructs(cfg.Fset, loaded)
	if err != nil {
		return err
	}

	sort.Slice(packagesToGenerate, func(i, j int) bool {
		return packagesToGenerate[i].dir < packagesToGenerate[j].dir
	})

	for _, pkg := range packagesToGenerate {
		content, err := generatePackage(pkg, generatedTypes)
		if err != nil {
			return fmt.Errorf("generate package %s: %w", pkg.types.Path(), err)
		}
		if err := writeFile(filepath.Join(pkg.dir, generatedFile), content); err != nil {
			return err
		}
	}

	return nil
}

func generatedFilesOverlay(root string) (map[string][]byte, error) {
	overlay := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if entry.Name() != generatedFile {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.PackageClauseOnly)
		if err != nil {
			return fmt.Errorf("read package clause from %s: %w", path, err)
		}
		overlay[path] = []byte("package " + parsed.Name.Name + "\n")
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan existing generated files: %w", err)
	}
	return overlay, nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("find go.mod: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in the current directory or its parents")
		}
		dir = parent
	}
}

func findResetStructs(fset *token.FileSet, loaded []*packages.Package) ([]packageToGenerate, map[*types.Named]struct{}, error) {
	result := make([]packageToGenerate, 0)
	generatedTypes := make(map[*types.Named]struct{})

	for _, loadedPkg := range loaded {
		if loadedPkg.Types == nil || loadedPkg.TypesInfo == nil {
			continue
		}

		pkg := packageToGenerate{
			name:  loadedPkg.Name,
			types: loadedPkg.Types,
			info:  loadedPkg.TypesInfo,
		}

		for _, file := range loadedPkg.Syntax {
			filename := fset.Position(file.Pos()).Filename
			if filepath.Base(filename) == generatedFile {
				continue
			}

			for _, declaration := range file.Decls {
				genDecl, ok := declaration.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !markedForReset(genDecl, typeSpec) {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						return nil, nil, fmt.Errorf("%s: %s is marked with %s but is not a struct", filename, typeSpec.Name.Name, directive)
					}

					object, ok := loadedPkg.TypesInfo.Defs[typeSpec.Name].(*types.TypeName)
					if !ok {
						return nil, nil, fmt.Errorf("%s: cannot resolve type %s", filename, typeSpec.Name.Name)
					}
					named, ok := types.Unalias(object.Type()).(*types.Named)
					if !ok {
						return nil, nil, fmt.Errorf("%s: cannot resolve named struct %s", filename, typeSpec.Name.Name)
					}

					pkg.dir = filepath.Dir(filename)
					pkg.structs = append(pkg.structs, resetStruct{
						name:       typeSpec.Name.Name,
						receiver:   receiverType(typeSpec),
						structType: structType,
					})
					generatedTypes[named.Origin()] = struct{}{}
				}
			}
		}

		if len(pkg.structs) != 0 {
			sort.Slice(pkg.structs, func(i, j int) bool {
				return pkg.structs[i].name < pkg.structs[j].name
			})
			result = append(result, pkg)
		}
	}

	return result, generatedTypes, nil
}

func markedForReset(declaration *ast.GenDecl, spec *ast.TypeSpec) bool {
	if hasDirective(spec.Doc) {
		return true
	}
	return len(declaration.Specs) == 1 && hasDirective(declaration.Doc)
}

func hasDirective(group *ast.CommentGroup) bool {
	if group == nil {
		return false
	}
	for _, comment := range group.List {
		if strings.TrimSpace(comment.Text) == directive {
			return true
		}
	}
	return false
}

func receiverType(spec *ast.TypeSpec) string {
	if spec.TypeParams == nil || len(spec.TypeParams.List) == 0 {
		return spec.Name.Name
	}

	parameters := make([]string, 0, len(spec.TypeParams.List))
	for _, field := range spec.TypeParams.List {
		for _, name := range field.Names {
			parameters = append(parameters, name.Name)
		}
	}
	return spec.Name.Name + "[" + strings.Join(parameters, ", ") + "]"
}

func generatePackage(pkg packageToGenerate, generatedTypes map[*types.Named]struct{}) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by reset; DO NOT EDIT.")
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "package %s\n", pkg.name)

	for _, structure := range pkg.structs {
		fmt.Fprintln(&output)
		fmt.Fprintf(&output, "func (x *%s) Reset() {\n", structure.receiver)
		fmt.Fprintln(&output, "\tif x == nil {")
		fmt.Fprintln(&output, "\t\treturn")
		fmt.Fprintln(&output, "\t}")

		for _, field := range structure.structType.Fields.List {
			fieldType := pkg.info.TypeOf(field.Type)
			if fieldType == nil {
				return nil, fmt.Errorf("cannot resolve a field type in %s", structure.name)
			}

			for _, fieldName := range fieldNames(field) {
				generateReset(&output, "x."+fieldName, fieldType, pkg.types, generatedTypes, 1)
			}
		}

		fmt.Fprintln(&output, "}")
	}

	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated code: %w\n%s", err, output.String())
	}
	return formatted, nil
}

func fieldNames(field *ast.Field) []string {
	if len(field.Names) != 0 {
		result := make([]string, 0, len(field.Names))
		for _, name := range field.Names {
			if name.Name != "_" {
				result = append(result, name.Name)
			}
		}
		return result
	}

	if name := embeddedFieldName(field.Type); name != "" {
		return []string{name}
	}
	return nil
}

func embeddedFieldName(expression ast.Expr) string {
	switch fieldType := expression.(type) {
	case *ast.Ident:
		return fieldType.Name
	case *ast.SelectorExpr:
		return fieldType.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(fieldType.X)
	case *ast.IndexExpr:
		return embeddedFieldName(fieldType.X)
	case *ast.IndexListExpr:
		return embeddedFieldName(fieldType.X)
	case *ast.ParenExpr:
		return embeddedFieldName(fieldType.X)
	}
	return ""
}

func generateReset(output *bytes.Buffer, expression string, fieldType types.Type, pkg *types.Package, generatedTypes map[*types.Named]struct{}, level int) {
	indent := strings.Repeat("\t", level)
	fieldType = types.Unalias(fieldType)
	if _, ok := fieldType.(*types.TypeParam); ok {
		fmt.Fprintf(output, "%s{\n", indent)
		fmt.Fprintf(output, "%s\tvar zero %s\n", indent, types.TypeString(fieldType, nil))
		fmt.Fprintf(output, "%s\t%s = zero\n", indent, expression)
		fmt.Fprintf(output, "%s}\n", indent)
		return
	}
	underlying := fieldType.Underlying()

	if pointer, ok := underlying.(*types.Pointer); ok {
		fmt.Fprintf(output, "%sif %s != nil {\n", indent, expression)
		generateReset(output, "*"+parenthesize(expression), pointer.Elem(), pkg, generatedTypes, level+1)
		fmt.Fprintf(output, "%s}\n", indent)
		return
	}

	if _, ok := underlying.(*types.Interface); ok {
		fmt.Fprintf(output, "%sif resetter, ok := %s.(interface{ Reset() }); ok {\n", indent, parenthesize(expression))
		fmt.Fprintf(output, "%s\tresetter.Reset()\n", indent)
		fmt.Fprintf(output, "%s}\n", indent)
		return
	}

	if resetCall, ok := concreteResetCall(expression, fieldType, pkg, generatedTypes); ok {
		fmt.Fprintf(output, "%s%s\n", indent, resetCall)
		return
	}

	switch typed := underlying.(type) {
	case *types.Basic:
		switch {
		case typed.Info()&types.IsBoolean != 0:
			fmt.Fprintf(output, "%s%s = false\n", indent, expression)
		case typed.Info()&types.IsString != 0:
			fmt.Fprintf(output, "%s%s = \"\"\n", indent, expression)
		case typed.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			fmt.Fprintf(output, "%s%s = 0\n", indent, expression)
		case typed.Kind() == types.UnsafePointer:
			fmt.Fprintf(output, "%s%s = nil\n", indent, expression)
		}
	case *types.Slice:
		fmt.Fprintf(output, "%s%s = %s[:0]\n", indent, expression, parenthesize(expression))
	case *types.Map:
		fmt.Fprintf(output, "%sclear(%s)\n", indent, expression)
	case *types.Array:
		fmt.Fprintf(output, "%sfor i := range %s {\n", indent, expression)
		generateReset(output, parenthesize(expression)+"[i]", typed.Elem(), pkg, generatedTypes, level+1)
		fmt.Fprintf(output, "%s}\n", indent)
	case *types.Chan, *types.Signature:
		fmt.Fprintf(output, "%s%s = nil\n", indent, expression)
	case *types.Struct:
	}
}

func concreteResetCall(expression string, fieldType types.Type, pkg *types.Package, generatedTypes map[*types.Named]struct{}) (string, bool) {
	if named := namedStruct(fieldType); named != nil {
		if _, ok := generatedTypes[named.Origin()]; ok {
			return parenthesize(expression) + ".Reset()", true
		}
	}

	if _, ok := fieldType.Underlying().(*types.Struct); !ok {
		return "", false
	}
	if hasResetMethod(fieldType, pkg) {
		return parenthesize(expression) + ".Reset()", true
	}
	if hasResetMethod(types.NewPointer(fieldType), pkg) {
		return parenthesize(expression) + ".Reset()", true
	}
	return "", false
}

func namedStruct(fieldType types.Type) *types.Named {
	fieldType = types.Unalias(fieldType)
	if pointer, ok := fieldType.(*types.Pointer); ok {
		fieldType = types.Unalias(pointer.Elem())
	}
	named, ok := fieldType.(*types.Named)
	if !ok {
		return nil
	}
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return nil
	}
	return named
}

func hasResetMethod(fieldType types.Type, pkg *types.Package) bool {
	object, _, _ := types.LookupFieldOrMethod(fieldType, true, pkg, "Reset")
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	return ok && signature.Params().Len() == 0 && signature.Results().Len() == 0
}

func parenthesize(expression string) string {
	if strings.HasPrefix(expression, "*") {
		return "(" + expression + ")"
	}
	return expression
}

func writeFile(filename string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".reset-*.go")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", filename, err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", filename, err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set permissions for %s: %w", filename, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", filename, err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("replace %s: %w", filename, err)
	}
	return nil
}
