package types

import (
	"math/rand"
	"strings"
	"sync"
)

type MarkovChain struct {
	Chain map[string][]string
	Mu    sync.RWMutex
}

func NewMarkovChain() *MarkovChain {
	return &MarkovChain{
		Chain: make(map[string][]string),
	}
}

func (m *MarkovChain) Train(text string) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	words := strings.Fields(text)
	if len(words) < 2 {
		return
	}

	for i := 0; i < len(words)-1; i++ {
		word := strings.ToLower(words[i])
		next := words[i+1]
		m.Chain[word] = append(m.Chain[word], next)
	}
}

func (m *MarkovChain) Generate(maxWords int) string {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	if len(m.Chain) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m.Chain))
	for k := range m.Chain {
		keys = append(keys, k)
	}

	current := keys[rand.Intn(len(keys))]
	result := []string{current}

	for i := 1; i < maxWords; i++ {
		next, ok := m.Chain[strings.ToLower(current)]
		if !ok || len(next) == 0 {
			break
		}
		current = next[rand.Intn(len(next))]
		result = append(result, current)
	}

	return strings.Join(result, " ")
}
