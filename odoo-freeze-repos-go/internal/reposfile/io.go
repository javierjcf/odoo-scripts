// Package reposfile edita repos.yaml línea a línea. No se re-serializa el YAML: se
// modifican solo las líneas necesarias para conservar comentarios, orden y formato.
package reposfile

import (
	"io"
	"os"
	"strings"
)

// ReadLines devuelve las líneas conservando el salto final de cada una.
func ReadLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	lines := strings.SplitAfter(string(raw), "\n")
	if lines[len(lines)-1] == "" { // texto acabado en "\n" deja un elemento vacío final
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// WriteLines escribe las líneas manteniendo los permisos del fichero si ya existía.
func WriteLines(path string, lines []string) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "")), mode)
}

// CopyFile copia src a dst con los mismos permisos.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() // defer: se ejecuta al salir de la función, pase lo que pase

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// FileExists indica si la ruta existe.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
