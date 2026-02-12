package filter

import (
	"sync"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// AhoCorasickMatch represents a match found by Aho-Corasick algorithm.
type AhoCorasickMatch struct {
	Word      string
	Position  int
	Category  string
	NsfwScore float64
}

// PatternInfo stores metadata about a pattern.
type PatternInfo struct {
	Word      string
	Category  string
	NsfwScore float64
}

// ahoCorasickNode represents a node in the Aho-Corasick automaton.
type ahoCorasickNode struct {
	children    map[rune]*ahoCorasickNode
	failLink    *ahoCorasickNode
	output      []PatternInfo
	isEndOfWord bool
}

// AhoCorasick implements the Aho-Corasick string matching algorithm.
type AhoCorasick struct {
	root *ahoCorasickNode
	mu   sync.RWMutex
}

// NewAhoCorasick creates a new Aho-Corasick automaton.
func NewAhoCorasick() *AhoCorasick {
	return &AhoCorasick{
		root: newAhoCorasickNode(),
	}
}

func newAhoCorasickNode() *ahoCorasickNode {
	return &ahoCorasickNode{
		children: make(map[rune]*ahoCorasickNode),
		output:   make([]PatternInfo, 0),
	}
}

// Build builds the automaton from a list of patterns.
func (ac *AhoCorasick) Build(patterns []PatternInfo) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Reset the automaton
	ac.root = newAhoCorasickNode()

	// Build the trie
	for _, pattern := range patterns {
		ac.addPattern(pattern)
	}

	// Build fail links using BFS
	ac.buildFailLinks()
}

// addPattern adds a single pattern to the trie.
func (ac *AhoCorasick) addPattern(pattern PatternInfo) {
	node := ac.root
	normalizedWord := NormalizeText(pattern.Word)

	for _, char := range normalizedWord {
		if _, ok := node.children[char]; !ok {
			node.children[char] = newAhoCorasickNode()
		}
		node = node.children[char]
	}

	node.isEndOfWord = true
	node.output = append(node.output, pattern)
}

// buildFailLinks builds the fail links for the automaton.
func (ac *AhoCorasick) buildFailLinks() {
	queue := make([]*ahoCorasickNode, 0)

	// Initialize fail links for depth 1 nodes
	for _, child := range ac.root.children {
		child.failLink = ac.root
		queue = append(queue, child)
	}

	// BFS to build fail links for deeper nodes
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for char, child := range current.children {
			queue = append(queue, child)

			// Find the longest proper suffix that is also a prefix
			failNode := current.failLink
			for failNode != nil && failNode.children[char] == nil {
				failNode = failNode.failLink
			}

			if failNode == nil {
				child.failLink = ac.root
			} else {
				child.failLink = failNode.children[char]
				// Merge output from fail link
				child.output = append(child.output, child.failLink.output...)
			}
		}
	}
}

// Search searches for all patterns in the given text.
func (ac *AhoCorasick) Search(text string) []AhoCorasickMatch {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	matches := make([]AhoCorasickMatch, 0)
	normalizedText := NormalizeText(text)
	node := ac.root
	position := 0

	for _, char := range normalizedText {
		// Follow fail links until we find a match or reach root
		for node != nil && node.children[char] == nil {
			node = node.failLink
		}

		if node == nil {
			node = ac.root
		} else {
			node = node.children[char]
		}

		// Check for matches at this position
		for _, pattern := range node.output {
			matches = append(matches, AhoCorasickMatch{
				Word:      pattern.Word,
				Position:  position - len([]rune(pattern.Word)) + 1,
				Category:  pattern.Category,
				NsfwScore: pattern.NsfwScore,
			})
		}
		position++
	}

	return matches
}

// HasMatch checks if any pattern matches the text (faster than Search).
func (ac *AhoCorasick) HasMatch(text string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	normalizedText := NormalizeText(text)
	node := ac.root

	for _, char := range normalizedText {
		for node != nil && node.children[char] == nil {
			node = node.failLink
		}

		if node == nil {
			node = ac.root
		} else {
			node = node.children[char]
		}

		if len(node.output) > 0 {
			return true
		}
	}

	return false
}

// NormalizeText normalizes text for matching.
// - Converts to lowercase
// - Removes diacritics
// - Normalizes unicode
// - Handles leetspeak
func NormalizeText(text string) string {
	// Normalize unicode
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)), // Remove diacritics
		norm.NFC,
	)
	result, _, _ := transform.String(t, text)

	// Convert to lowercase
	lowered := make([]rune, 0, len(result))
	for _, r := range result {
		lowered = append(lowered, unicode.ToLower(r))
	}

	// Handle leetspeak substitutions
	leetMap := map[rune]rune{
		'0': 'o',
		'1': 'i',
		'3': 'e',
		'4': 'a',
		'5': 's',
		'7': 't',
		'8': 'b',
		'@': 'a',
		'$': 's',
	}

	normalized := make([]rune, 0, len(lowered))
	for _, r := range lowered {
		if replacement, ok := leetMap[r]; ok {
			normalized = append(normalized, replacement)
		} else {
			normalized = append(normalized, r)
		}
	}

	return string(normalized)
}
