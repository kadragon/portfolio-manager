// Package arch holds architectural boundary tests enforcing layer separation,
// the Go equivalent of tests/arch/test_layer_boundaries.py. They encode Golden
// Principles 1 and 3 from AGENTS.md:
//
//	GP-1: only the repository layer accesses the database (internal/db, internal/db/sqlc).
//	GP-3: web -> services -> repositories -> db; reverse imports are violations.
package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/kadragon/portfolio-manager"

// layerImports returns every imported package path for .go files under
// internal/<layer> (recursively), keyed by the file's repo-relative path.
func layerImports(t *testing.T, layer string) map[string][]string {
	t.Helper()
	root := repoRoot(t)
	dir := filepath.Join(root, "internal", layer)
	result := map[string][]string{}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return result // layer not built yet — nothing to enforce
	}
	fset := token.NewFileSet()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Test files legitimately construct a database (e.g. db.OpenMemory) for
		// fixtures; the Python arch test does not see them because Python tests
		// live in a separate tests/ tree. Exclude them to keep parity.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			result[rel] = append(result[rel], p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return result
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot locate repo root")
		}
		dir = parent
	}
}

func assertNoImport(t *testing.T, layer, forbidden, principle string) {
	t.Helper()
	for file, imports := range layerImports(t, layer) {
		for _, imp := range imports {
			if strings.HasPrefix(imp, forbidden) {
				t.Errorf("%s imports %q (%s)", file, imp, principle)
			}
		}
	}
}

// GP-1: the database is accessed only by the repository layer.
func TestWebHasNoDirectDBAccess(t *testing.T) {
	assertNoImport(t, "web", modulePath+"/internal/db", "GP-1: use a repository, not the DB directly")
}

func TestServicesHaveNoDirectDBAccess(t *testing.T) {
	assertNoImport(t, "services", modulePath+"/internal/db", "GP-1: use a repository, not the DB directly")
}

// GP-3: layer dependency direction.
func TestServicesDoNotImportWeb(t *testing.T) {
	assertNoImport(t, "services", modulePath+"/internal/web", "GP-3: services must not depend on web")
}

func TestRepositoriesDoNotImportWeb(t *testing.T) {
	assertNoImport(t, "repositories", modulePath+"/internal/web", "GP-3: repositories must not depend on web")
}

func TestRepositoriesDoNotImportServices(t *testing.T) {
	assertNoImport(t, "repositories", modulePath+"/internal/services", "GP-3: repositories must not depend on services")
}
