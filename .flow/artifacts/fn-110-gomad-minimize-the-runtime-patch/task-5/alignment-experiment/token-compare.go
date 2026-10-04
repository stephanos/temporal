package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

type scannedToken struct {
	Kind    token.Token
	Literal string
}

type scanResult struct {
	Tokens   []scannedToken
	Comments []string
}

func scan(path string) (scanResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return scanResult{}, err
	}
	if bytes.ContainsRune(data, '\r') {
		return scanResult{}, fmt.Errorf("CR bytes would need a separate raw-comment check: %s", path)
	}
	fset := token.NewFileSet()
	var s scanner.Scanner
	var scanErr error
	s.Init(fset.AddFile(path, fset.Base(), len(data)), data, func(p token.Position, message string) {
		scanErr = fmt.Errorf("%s: %s", p, message)
	}, scanner.ScanComments)
	var result scanResult
	for {
		_, kind, literal := s.Scan()
		result.Tokens = append(result.Tokens, scannedToken{kind, literal})
		if kind == token.COMMENT {
			result.Comments = append(result.Comments, literal)
		}
		if kind == token.EOF {
			break
		}
	}
	return result, scanErr
}

func digest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func compare(before, after string) error {
	var names []string
	err := filepath.WalkDir(before, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".go" {
			name, err := filepath.Rel(before, path)
			if err != nil {
				return err
			}
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(names)
	var rows []map[string]any
	for _, name := range names {
		left, err := scan(filepath.Join(before, name))
		if err != nil {
			return err
		}
		right, err := scan(filepath.Join(after, name))
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(left, right) {
			return fmt.Errorf("token/comment sequence differs: %s", name)
		}
		rows = append(rows, map[string]any{
			"path": name, "tokens": len(left.Tokens), "comments": len(left.Comments),
			"token_sequence_sha256": digest(left.Tokens), "comment_sequence_sha256": digest(left.Comments),
			"tokens_equal_including_comments_and_inserted_semicolons": true,
			"comment_literal_bytes_equal": true,
		})
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"files": rows, "go_files": len(rows), "all_equal": true})
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: token-compare BEFORE AFTER")
		os.Exit(2)
	}
	if err := compare(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
