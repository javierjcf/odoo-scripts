package diff

import (
	"strings"
	"testing"
)

func TestUnifiedDiff(t *testing.T) {
	a := []string{"1\n", "2\n", "3\n", "4\n", "5\n"}
	b := []string{"1\n", "2\n", "tres\n", "4\n", "5\n", "6\n"}

	var sb strings.Builder
	Unified(&sb, "a", "b", a, b, 1)
	want := `--- a
+++ b
@@ -2,4 +2,5 @@
 2
-3
+tres
 4
 5
+6
`
	if sb.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", sb.String(), want)
	}
}

func TestUnifiedDiffNoChanges(t *testing.T) {
	var sb strings.Builder
	Unified(&sb, "a", "b", []string{"x\n"}, []string{"x\n"}, 3)
	if sb.Len() != 0 {
		t.Errorf("no deberia imprimir nada, got %q", sb.String())
	}
}
