// Package gitx envuelve las llamadas a la CLI de git y la resolución de refs a commits.
package gitx

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunGit ejecuta `git -C repoPath args...` y devuelve la salida sin espacios finales.
// En Go no hay excepciones: se devuelve (valor, error) y quien llama decide qué hacer.
func RunGit(repoPath string, args ...string) (string, error) {
	full := append([]string{"-C", repoPath}, args...)
	out, err := exec.Command("git", full...).Output()
	if err != nil {
		// Si git falló, Output() devuelve *exec.ExitError con el stderr dentro.
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsSHA indica si ref es un hash completo de 40 caracteres hexadecimales.
func IsSHA(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range strings.ToLower(ref) {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// ResolveRef resuelve (remote, ref) al commit que representa hoy, sin red.
//
// Dos estrategias, en orden:
//  1. Rama normal ya trackeada localmente (<remote>/<rama>): exacta y sin ambigüedad.
//  2. Ref puntual (PR, etc.) que git-aggregator fusionó en su día: se busca el merge
//     commit cuyo mensaje menciona esa ref exacta ("Merge commit '<ref>' of ... into
//     ...") y se toma su segundo padre. Se usa el más reciente si hay varios.
func ResolveRef(repoPath, remote, ref, odooVersion string) (string, error) {
	branch := strings.ReplaceAll(ref, "$ODOO_VERSION", odooVersion)
	if sha, err := RunGit(repoPath, "rev-parse", "--verify", "--quiet", remote+"/"+branch); err == nil {
		return sha, nil
	}

	log, err := RunGit(repoPath, "log", "--first-parent", "--merges", "--format=%H\t%s")
	if err != nil {
		return "", fmt.Errorf("error ejecutando git log (%w)", err)
	}
	for _, line := range strings.Split(log, "\n") {
		sha, subject, _ := strings.Cut(line, "\t")
		if strings.Contains(subject, "'"+ref+"'") {
			parent, err := RunGit(repoPath, "rev-parse", sha+"^2")
			if err != nil {
				return "", fmt.Errorf("no se pudo resolver el segundo padre de %.10s (%w)", sha, err)
			}
			return parent, nil
		}
	}
	return "", fmt.Errorf("no se encontro ningun merge commit que mencione '%s'", ref)
}
