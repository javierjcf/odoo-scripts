// Package diff implementa un diff unificado mínimo para --dry-run (la librería estándar
// de Go no incluye diff).
package diff

import (
	"fmt"
	"io"
	"strings"
)

type diffOp struct {
	kind byte // ' ' igual, '-' borrada, '+' añadida
	text string
}

// diffLines calcula las operaciones con LCS (subsecuencia común más larga).
// Antes recorta prefijo y sufijo comunes: en repos.yaml casi todo es igual.
func diffLines(a, b []string) []diffOp {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	am, bm := a[pre:len(a)-suf], b[pre:len(b)-suf]

	ops := make([]diffOp, 0, len(a)+len(b))
	for _, l := range a[:pre] {
		ops = append(ops, diffOp{' ', l})
	}

	n, m := len(am), len(bm)
	dp := make([]int, (n+1)*(m+1)) // tabla aplanada: dp[i][j] -> dp[i*(m+1)+j]
	at := func(i, j int) int { return i*(m+1) + j }
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if am[i] == bm[j] {
				dp[at(i, j)] = dp[at(i+1, j+1)] + 1
			} else {
				dp[at(i, j)] = max(dp[at(i+1, j)], dp[at(i, j+1)])
			}
		}
	}
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case am[i] == bm[j]:
			ops = append(ops, diffOp{' ', am[i]})
			i++
			j++
		case dp[at(i+1, j)] >= dp[at(i, j+1)]:
			ops = append(ops, diffOp{'-', am[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', bm[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', am[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', bm[j]})
	}

	for _, l := range a[len(a)-suf:] {
		ops = append(ops, diffOp{' ', l})
	}
	return ops
}

// Unified imprime el diff en formato unificado con `ctx` líneas de contexto.
// No imprime nada si no hay diferencias.
func Unified(w io.Writer, oldName, newName string, a, b []string, ctx int) {
	ops := diffLines(a, b)

	// oldAt[i] / newAt[i]: líneas de cada versión consumidas antes de la operación i.
	oldAt := make([]int, len(ops)+1)
	newAt := make([]int, len(ops)+1)
	for i, op := range ops {
		oldAt[i+1], newAt[i+1] = oldAt[i], newAt[i]
		if op.kind != '+' {
			oldAt[i+1]++
		}
		if op.kind != '-' {
			newAt[i+1]++
		}
	}

	headerDone := false
	for i := 0; i < len(ops); {
		if ops[i].kind == ' ' {
			i++
			continue
		}
		// Un hunk empieza en el primer cambio y se extiende mientras el siguiente cambio
		// esté a menos de 2*ctx líneas iguales.
		last := i
		for k := i + 1; k < len(ops); k++ {
			if ops[k].kind != ' ' {
				last = k
			} else if k-last-1 > 2*ctx {
				break
			}
		}
		start, end := max(0, i-ctx), min(len(ops), last+ctx+1)

		if !headerDone {
			fmt.Fprintf(w, "--- %s\n+++ %s\n", oldName, newName)
			headerDone = true
		}
		oldCount := oldAt[end] - oldAt[start]
		newCount := newAt[end] - newAt[start]
		oldStart, newStart := oldAt[start]+1, newAt[start]+1
		if oldCount == 0 {
			oldStart--
		}
		if newCount == 0 {
			newStart--
		}
		fmt.Fprintf(w, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, op := range ops[start:end] {
			fmt.Fprintf(w, "%c%s\n", op.kind, strings.TrimRight(op.text, "\r\n"))
		}
		i = end
	}
}
