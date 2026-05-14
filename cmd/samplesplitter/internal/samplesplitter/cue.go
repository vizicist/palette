package samplesplitter

type CueData struct {
	File       string    `json:"file"`
	Duration   float64   `json:"duration"`
	Mode       string    `json:"mode"`
	Splits     []float64 `json:"splits"`
	PeakStarts []float64 `json:"peak_starts"`
	NumSplits  int       `json:"num_splits"`
	Words      *int      `json:"words_per_split,omitempty"`
}

type AnalyzeOptions struct {
	Mode             string
	Interval         float64
	SilenceThreshold float64
	SilenceMinimum   float64
	WordsPerSplit    int
}

func DefaultAnalyzeOptions() AnalyzeOptions {
	return AnalyzeOptions{
		Mode:             DefaultSplitMode,
		Interval:         DefaultIntervalSeconds,
		SilenceThreshold: DefaultSilenceThreshold,
		SilenceMinimum:   DefaultSilenceMinimum,
		WordsPerSplit:    DefaultWordsPerSplit,
	}
}
