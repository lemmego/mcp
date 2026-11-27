package mcp

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type reviewCodeInput struct {
	Root string `json:"root,omitempty" jsonschema:"project root to inspect; defaults to the current working directory"`
}

type reviewCodeFinding struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type reviewCodeOutput struct {
	Root     string              `json:"root"`
	Findings []reviewCodeFinding `json:"findings"`
}

var handlerActionSuffixes = []string{
	"IndexHandler", "CreateHandler", "ShowHandler", "StoreHandler",
	"EditHandler", "UpdateHandler", "DeleteHandler", "InvokeHandler",
}

var reviewLayerFiles = map[string]func(string) bool{
	"handlers": func(name string) bool {
		return name == "doc.go" || strings.HasSuffix(name, "_handlers.go") || strings.HasSuffix(name, "_handler.go")
	},
	"models": func(name string) bool { return strings.HasSuffix(name, ".go") },
	"repos": func(name string) bool {
		return strings.HasSuffix(name, "_repo.go") || strings.HasSuffix(name, "_repository.go")
	},
	"services": func(name string) bool { return name == "doc.go" || strings.HasSuffix(name, "_service.go") },
	"inputs":   func(name string) bool { return strings.HasSuffix(name, "_input.go") },
}

var standardInternalDirectories = map[string]bool{
	"handlers": true, "models": true, "repos": true, "repositories": true,
	"services": true, "inputs": true, "routes": true, "middleware": true,
	"migrations": true, "commands": true, "configs": true, "plugins": true,
	"policies": true, "views": true, "providers": true,
}

func (s *Server) addReviewCode(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "review-code",
		Description: "Check a Lemmego project's predictable internal layout: resourceful versus invokable handler names, " +
			"one handler kind per file, helper functions kept out of handler files, 20-line function limits in " +
			"handlers/models/repos/services/inputs, layer-specific filenames, and custom internal directories. " +
			"Pass a project root or omit it to inspect the current working directory. This reports findings; it does not edit files.",
	}, s.reviewCode)
}

func (s *Server) reviewCode(_ context.Context, _ *sdk.CallToolRequest, in reviewCodeInput) (*sdk.CallToolResult, reviewCodeOutput, error) {
	root := strings.TrimSpace(in.Root)
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return failure("cannot determine project root: %v", err), reviewCodeOutput{}, nil
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return failure("cannot resolve project root: %v", err), reviewCodeOutput{}, nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return failure("project root %q is not a readable directory", root), reviewCodeOutput{}, nil
	}

	out := reviewCodeOutput{Root: root, Findings: []reviewCodeFinding{}}
	internal := filepath.Join(root, "internal")
	if _, err := os.Stat(internal); err == nil {
		s.reviewInternalDirectories(root, internal, &out)
		for layer := range reviewLayerFiles {
			layerPath := filepath.Join(internal, layer)
			if _, err := os.Stat(layerPath); err == nil {
				s.reviewLayer(root, layerPath, layer, &out)
			}
		}
	}
	sort.Slice(out.Findings, func(i, j int) bool {
		if out.Findings[i].Path != out.Findings[j].Path {
			return out.Findings[i].Path < out.Findings[j].Path
		}
		if out.Findings[i].Line != out.Findings[j].Line {
			return out.Findings[i].Line < out.Findings[j].Line
		}
		return out.Findings[i].Rule < out.Findings[j].Rule
	})

	var b strings.Builder
	if len(out.Findings) == 0 {
		fmt.Fprintf(&b, "review-code: no findings in %s\n", root)
	} else {
		fmt.Fprintf(&b, "review-code: %d finding(s) in %s\n", len(out.Findings), root)
		for _, finding := range out.Findings {
			if finding.Line > 0 {
				fmt.Fprintf(&b, "  %s:%d [%s] %s\n", finding.Path, finding.Line, finding.Rule, finding.Message)
			} else {
				fmt.Fprintf(&b, "  %s [%s] %s\n", finding.Path, finding.Rule, finding.Message)
			}
		}
	}
	return text(b.String()), out, nil
}

func (s *Server) reviewInternalDirectories(root, internal string, out *reviewCodeOutput) {
	entries, err := os.ReadDir(internal)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && !standardInternalDirectories[entry.Name()] {
			addReviewFinding(root, out, filepath.Join(internal, entry.Name()), 0, "internal-directory",
				"custom internal package directory; move this code into internal/services/ or an existing layer directory")
		}
	}
}

func (s *Server) reviewLayer(root, dir, layer string, out *reviewCodeOutput) {
	walker := func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			addReviewFinding(root, out, path, 0, "read-error", walkErr.Error())
			return nil
		}
		if info.IsDir() {
			if path != dir {
				addReviewFinding(root, out, path, 0, "nested-layer-directory",
					"keep layer files directly in internal/"+layer+"; move reusable behavior to internal/services/")
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".go") {
			addReviewFinding(root, out, path, 0, "layer-file",
				"layer directories contain Go source and tests only; move this file to the appropriate package")
			return nil
		}
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if !reviewLayerFiles[layer](name) {
			addReviewFinding(root, out, path, 0, "layer-file-name",
				"filename does not match the "+layer+" layer convention; use a layer-specific filename and keep tests in *_test.go")
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			addReviewFinding(root, out, path, 0, "parse-error", err.Error())
			return nil
		}
		if layer == "handlers" {
			reviewHandlerFile(root, path, fset, file, out)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}
			start := fset.Position(fn.Pos()).Line
			end := fset.Position(fn.End()).Line
			if end-start+1 > 20 {
				addReviewFinding(root, out, path, start, "function-length",
					fmt.Sprintf("%s is %d lines; refactor functions in internal/%s to 20 lines or fewer", fn.Name.Name, end-start+1, layer))
			}
			return false
		})
		return nil
	}
	_ = filepath.Walk(dir, walker)
}

func reviewHandlerFile(root, path string, fset *token.FileSet, file *ast.File, out *reviewCodeOutput) {
	resourceful := 0
	invokable := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		allowed := false
		for _, suffix := range handlerActionSuffixes {
			if strings.HasSuffix(fn.Name.Name, suffix) {
				allowed = true
				if suffix == "InvokeHandler" {
					invokable++
				} else {
					resourceful++
				}
				break
			}
		}
		line := fset.Position(fn.Pos()).Line
		if !allowed {
			addReviewFinding(root, out, path, line, "handler-method",
				fmt.Sprintf("%s is not a resourceful action or InvokeHandler; move helpers to models, repos, or services", fn.Name.Name))
		}
	}
	if resourceful > 0 && invokable > 0 {
		addReviewFinding(root, out, path, 0, "handler-kind",
			"handler file mixes resourceful actions and InvokeHandler; split it into resourceful and invokable handler files")
	}
	if invokable > 1 {
		addReviewFinding(root, out, path, 0, "handler-kind",
			"an invokable handler file must contain exactly one InvokeHandler method")
	}
}

func addReviewFinding(root string, out *reviewCodeOutput, path string, line int, rule, message string) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	out.Findings = append(out.Findings, reviewCodeFinding{
		Path: filepath.ToSlash(rel), Line: line, Rule: rule, Message: message,
	})
}
