// Package treecheck helps to check that a change to the parsers does not change
// the shape of the trees they build.
//
// It is meant for characterization tests: the digest of every tree is stored in
// a golden file, written once with -update on the code to preserve, and compared
// on the code after the change.
package treecheck

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LorenzoTavani/gopapageno"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// maxNodes stops the visit of a broken tree that contains a cycle.
const maxNodes = 50_000_000

// Digest returns a digest of the shape of the tree rooted at root and its number
// of nodes. The shape is the type of every node and the nesting of the children;
// the values are not part of it.
func Digest(root *gopapageno.Token) (string, int) {
	h := sha256.New()
	nodes := 0

	var record [3]byte
	emit := func(kind byte, tokenType gopapageno.TokenType) {
		record[0], record[1], record[2] = kind, byte(tokenType), byte(tokenType>>8)
		h.Write(record[:])
	}

	var walk func(t *gopapageno.Token) bool
	walk = func(t *gopapageno.Token) bool {
		nodes++
		if nodes > maxNodes {
			return false
		}

		emit(0, t.Type)

		if t.Child == nil {
			return true
		}

		emit(1, 0)
		for c := t.Child; c != nil; c = c.Next {
			if !walk(c) {
				return false
			}
		}
		emit(2, 0)

		return true
	}

	if !walk(root) {
		return "LOOP", nodes
	}

	return hex.EncodeToString(h.Sum(nil))[:16], nodes
}

// Check compares lines with the golden file at path. When the tests run with
// -update, it rewrites the file instead.
func Check(t *testing.T, path string, lines []string) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d lines)", path, len(lines))

		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the golden file, run the test with -update on the code to preserve: %v", err)
	}

	want := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(want) != len(lines) {
		t.Errorf("the golden file has %d lines, the test produced %d", len(want), len(lines))
	}

	differences := 0
	for i := 0; i < len(lines) && i < len(want); i++ {
		if lines[i] == want[i] {
			continue
		}

		differences++
		if differences > 10 {
			t.Errorf("more than 10 lines differ, stopping here")
			break
		}
		t.Errorf("line %d differs:\n  got  %s\n  want %s", i+1, lines[i], want[i])
	}
}
