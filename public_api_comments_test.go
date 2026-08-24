package req_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

var englishWordPattern = regexp.MustCompile(`[A-Za-z]{2,}`)

var placeholderChinesePhrases = []string{
	"上面的英文说明",
	"上面英文说明",
	"相应配置",
	"相应功能",
	"相应操作",
	"对应的 HTTP 请求操作",
	"对应请求",
}

// TestPublicCallableCommentsAreBilingual keeps every exported callable API
// documented in both English and Chinese. It intentionally covers the root
// package and the public subpackages maintained by this repository.
//
// TestPublicCallableCommentsAreBilingual 确保所有公开可调用 API 都同时提供
// 英文和中文注释；检查范围包括根包以及本仓库维护的公开子包。
func TestPublicCallableCommentsAreBilingual(t *testing.T) {
	publicPackageDirs := []string{
		".",
		"http2",
		filepath.Join("pkg", "altsvc"),
		filepath.Join("pkg", "tls"),
	}

	var problems []string
	checked := 0
	for _, dir := range publicPackageDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read public package %s: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			fileChecked, fileProblems := inspectPublicCallables(path, file)
			checked += fileChecked
			problems = append(problems, fileProblems...)
		}
	}
	t.Logf("checked %d exported callable APIs", checked)

	if len(problems) == 0 {
		return
	}
	sort.Strings(problems)
	t.Fatalf("public callable comments must start with the symbol name and contain separate English and Chinese text:\n%s", strings.Join(problems, "\n"))
}

func inspectPublicCallables(path string, file *ast.File) (int, []string) {
	var problems []string
	checked := 0
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if !decl.Name.IsExported() || !exportedReceiver(decl.Recv) {
				continue
			}
			checked++
			if problem := bilingualCommentProblem(decl.Name.Name, decl.Doc); problem != "" {
				problems = append(problems, path+": "+decl.Name.Name+": "+problem)
			}
		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				if !typeSpec.Name.IsExported() {
					continue
				}
				doc := typeSpec.Doc
				if doc == nil {
					doc = decl.Doc
				}
				if _, ok := typeSpec.Type.(*ast.FuncType); ok {
					checked++
					if problem := bilingualCommentProblem(typeSpec.Name.Name, doc); problem != "" {
						problems = append(problems, path+": "+typeSpec.Name.Name+": "+problem)
					}
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, method := range iface.Methods.List {
					for _, name := range method.Names {
						if !name.IsExported() {
							continue
						}
						checked++
						methodDoc := method.Doc
						if methodDoc == nil {
							methodDoc = method.Comment
						}
						if problem := bilingualCommentProblem(name.Name, methodDoc); problem != "" {
							label := typeSpec.Name.Name + "." + name.Name
							problems = append(problems, path+": "+label+": "+problem)
						}
					}
				}
			}
		}
	}
	return checked, problems
}

func exportedReceiver(recv *ast.FieldList) bool {
	if recv == nil {
		return true
	}
	if len(recv.List) != 1 {
		return false
	}
	return ast.IsExported(receiverName(recv.List[0].Type))
}

func receiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return receiverName(expr.X)
	case *ast.IndexExpr:
		return receiverName(expr.X)
	case *ast.IndexListExpr:
		return receiverName(expr.X)
	default:
		return ""
	}
}

func bilingualCommentProblem(name string, doc *ast.CommentGroup) string {
	if doc == nil {
		return "missing GoDoc"
	}
	lines := commentLines(doc)
	if len(lines) == 0 {
		return "empty GoDoc"
	}
	if !strings.HasPrefix(lines[0], name+" ") && lines[0] != name {
		return "English first line does not start with the symbol name"
	}
	if containsHan(lines[0]) || !englishWordPattern.MatchString(strings.TrimPrefix(lines[0], name)) {
		return "first line is not an English description"
	}
	hasChinese := false
	for _, line := range lines[1:] {
		if !containsHan(line) {
			continue
		}
		hasChinese = true
		for _, phrase := range placeholderChinesePhrases {
			if strings.Contains(line, phrase) {
				return "Chinese description contains placeholder wording " + phrase
			}
		}
	}
	if !hasChinese {
		return "missing Chinese description"
	}
	return ""
}

func commentLines(doc *ast.CommentGroup) []string {
	var lines []string
	for _, comment := range doc.List {
		text := strings.TrimSpace(comment.Text)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
