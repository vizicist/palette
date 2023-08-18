package kit

type Kval struct {
	phr *Phrase
}

func NewPhraseVal(s string) Kval {
	p := NewPhrase(s)
	return Kval {
		phr: p,
	}
}
