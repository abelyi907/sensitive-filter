# CODEBUDDY.md

This file provides guidance to CodeBuddy Code when working with code in this repository.

## Commands

- Run all tests with coverage: `go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html`
- Run single test: `go test -v -run TestFunctionName`
- Build: `go build ./...`

## Architecture

**Module:** `github.com/abelyi907/sensitive-filter`

Sensitive word filtering library using a **Double Trie** (dictionary tree) data structure for O(n) time complexity matching.

### Core Types

- `TrieNode` — children map, isEnd bool, isHomophone bool
- `SensitiveWordChecker` — root *TrieNode, with three mode flags:
  - `homophoneMode` — enables 谐音 (homophone) matching via pinyin conversion
  - `deformMode` — detects words separated by special characters (e.g., "暴-力", "赌_博")
  - `martianMode` — normalizes internet slang (火星文, e.g., "卜"->"不", "稀饭"->"喜欢")

### Global Singleton

`var SensitiveChecker *SensitiveWordChecker` — pre-initialized global instance.

### Key Methods

- `New()` — creates a new checker
- `Insert(word)` — add a single word to the trie
- `Contains(text)` — returns bool
- `FindAll(text)` — returns all matches with positions
- `Replace(text, replacement)` — replaces all matches
- `LoadFromTextFile(filepath)` — loads words from a text file, enables hot reload
- `LoadFromFileByLine(filepath)` — loads words line-by-line, no hot reload

### Hot Reload

`LoadFromTextFile` spawns a background goroutine that checks the file every 2 seconds and reloads on changes.

## Important Constraints

- **Not thread-safe** — concurrent use requires external synchronization
- **Deform mode enabled by default** — special characters are ignored when matching
- **Hot reload delay is 2 seconds** — changes are not detected instantly
- Case-insensitive matching (all input converted to lowercase before matching)
- Supports Chinese and Unicode characters

## Dependencies

- `github.com/mozillazg/go-pinyin v0.21.0` — for pinyin conversion in homophone mode
- Go version: 1.25.0