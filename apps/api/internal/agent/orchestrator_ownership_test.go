package agent

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoAgentPackageHasNoLifecycleConsumer(t *testing.T) {
	dir := agentPackageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) == "consumer.go" {
			t.Fatal("agent consumer must not exist — Python owns lifecycle")
		}
	}
}

func TestServiceHasNoProcessRunMethod(t *testing.T) {
	dir := agentPackageDir(t)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs["agent"]
	if pkg == nil {
		t.Fatal("agent package not found")
	}
	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				return true
			}
			if fn.Name.Name == "ProcessRun" {
				t.Fatal("Service.ProcessRun must not exist — Go must not orchestrate lifecycle")
			}
			return true
		})
	}
}

func TestStartRunDoesNotEnqueueAnalysisQueue(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(agentPackageDir(t), "service.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	if strings.Contains(src, "AnalysisQueue") && strings.Contains(src, "Enqueue") {
		t.Fatal("StartRun must not enqueue analysis:run:queue")
	}
	if strings.Contains(src, "func (s *Service) ProcessRun") {
		t.Fatal("ProcessRun must be removed from Go service")
	}
}

func agentPackageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Dir(file)
}
