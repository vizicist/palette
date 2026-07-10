package samplesplitter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

type Analyzer struct {
	FFmpegPath string
}

const wordValleySplitRatio = 0.5

const wordLoudnessWindowSeconds = 0.01

var ErrBelowWordThreshold = errors.New("MP3 does not exceed the word threshold")

func (a Analyzer) AnalyzeFile(mp3Path string, opts AnalyzeOptions) (CueData, []float64, error) {
	if opts.Mode == "" {
		opts.Mode = DefaultSplitMode
	}
	if opts.Interval <= 0 {
		opts.Interval = DefaultIntervalSeconds
	}
	if opts.SilenceThreshold <= 0 {
		opts.SilenceThreshold = DefaultSilenceThreshold
	}
	if opts.SilenceMinimum <= 0 {
		opts.SilenceMinimum = DefaultSilenceMinimum
	}
	if opts.WordsPerSplit <= 0 {
		opts.WordsPerSplit = DefaultWordsPerSplit
	}

	ffmpeg := a.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	tmp, err := os.CreateTemp("", "samplesplitter-*.wav")
	if err != nil {
		return CueData{}, nil, err
	}
	wavPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(wavPath)

	cmd := exec.Command(ffmpeg, "-y", "-i", mp3Path, "-ar", "44100", "-ac", "1", wavPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return CueData{}, nil, fmt.Errorf("ffmpeg failed: %w: %s", err, string(output))
	}

	samples, frameRate, duration, err := readPCM16WAV(wavPath)
	if err != nil {
		return CueData{}, nil, err
	}
	maxRMS := maxWindowRMS(samples, frameRate, wordLoudnessWindowSeconds)
	threshold := clampWordThreshold(opts.WordThreshold)
	if !exceedsWordThreshold(maxRMS, threshold) {
		return CueData{}, nil, fmt.Errorf("%w: maximum RMS %.4f, threshold %.4f", ErrBelowWordThreshold, maxRMS, threshold)
	}

	waveform := computeWaveform(samples, WaveformPoints)
	var splits []float64
	var wordSplits []float64
	switch opts.Mode {
	case "silence":
		splits = detectSplitsSilence(samples, frameRate, duration, opts.SilenceThreshold, opts.SilenceMinimum)
	case "words":
		wordSplits = detectSplitsWords(samples, frameRate, duration, opts.SilenceThreshold, opts.SilenceMinimum, 0.16, 0.65)
		splits = groupWordSplits(wordSplits, opts.WordsPerSplit)
	default:
		splits = detectSplitsFixed(duration, opts.Interval)
		opts.Mode = "fixed"
	}

	peakStarts := computePeakStarts(samples, frameRate, splits, duration)
	var words *int
	if opts.Mode == "words" {
		peakStarts = computeFirstWordPeakStarts(samples, frameRate, splits, wordSplits, duration)
		wordsValue := opts.WordsPerSplit
		words = &wordsValue
	}

	cue := CueData{
		File:       mp3Path,
		Duration:   round4(duration),
		Mode:       opts.Mode,
		Splits:     splits,
		PeakStarts: peakStarts,
		MaxRMS:     round4(maxRMS),
		NumSplits:  len(splits),
		Words:      words,
	}
	return cue, waveform, nil
}

func readPCM16WAV(path string) ([]float64, int, float64, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	var riff [12]byte
	if _, err := io.ReadFull(f, riff[:]); err != nil {
		return nil, 0, 0, err
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, 0, 0, errors.New("not a RIFF/WAVE file")
	}

	var channels uint16
	var sampleRate uint32
	var bitsPerSample uint16
	var data []byte

	for {
		var header [8]byte
		if _, err := io.ReadFull(f, header[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, 0, 0, err
		}
		chunkID := string(header[0:4])
		chunkSize := binary.LittleEndian.Uint32(header[4:8])
		chunk := make([]byte, chunkSize)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return nil, 0, 0, err
		}
		if chunkSize%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return nil, 0, 0, err
			}
		}

		switch chunkID {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, 0, 0, errors.New("short fmt chunk")
			}
			audioFormat := binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(chunk[14:16])
			if audioFormat != 1 || bitsPerSample != 16 {
				return nil, 0, 0, fmt.Errorf("unsupported WAV format %d/%d-bit", audioFormat, bitsPerSample)
			}
		case "data":
			data = chunk
		}
	}

	if sampleRate == 0 || channels == 0 || bitsPerSample != 16 || len(data) == 0 {
		return nil, 0, 0, errors.New("missing WAV format or data")
	}

	bytesPerSample := int(bitsPerSample / 8)
	frameSize := int(channels) * bytesPerSample
	frames := len(data) / frameSize
	samples := make([]float64, frames)
	for i := 0; i < frames; i++ {
		offset := i * frameSize
		v := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
		samples[i] = float64(v) / 32768.0
	}
	duration := float64(frames) / float64(sampleRate)
	return samples, int(sampleRate), duration, nil
}

func computeWaveform(samples []float64, points int) []float64 {
	if points <= 0 {
		points = WaveformPoints
	}
	block := max(1, len(samples)/points)
	out := make([]float64, points)
	for i := 0; i < points; i++ {
		start := i * block
		end := min(len(samples), (i+1)*block)
		if start >= end {
			continue
		}
		var sum float64
		for _, sample := range samples[start:end] {
			sum += sample * sample
		}
		out[i] = math.Sqrt(sum / float64(end-start))
	}
	peak := 0.0
	for _, v := range out {
		peak = max(peak, v)
	}
	if peak == 0 {
		peak = 1
	}
	for i := range out {
		out[i] /= peak
	}
	return out
}

func computePeakStarts(samples []float64, frameRate int, splits []float64, duration float64) []float64 {
	peakStarts := make([]float64, 0, len(splits))
	total := len(samples)
	for i, start := range splits {
		end := duration
		if i+1 < len(splits) {
			end = splits[i+1]
		}
		startIdx := max(0, min(total, int(start*float64(frameRate))))
		endIdx := max(startIdx+1, min(total, int(end*float64(frameRate))))
		if startIdx >= total || startIdx >= endIdx {
			peakStarts = append(peakStarts, round4(start))
			continue
		}
		chunk := samples[startIdx:endIdx]
		peakOffset := 0
		peakValue := -1.0
		for idx, sample := range chunk {
			abs := math.Abs(sample)
			if abs > peakValue {
				peakValue = abs
				peakOffset = idx
			}
		}
		peakStarts = append(peakStarts, round4(float64(startIdx+peakOffset)/float64(frameRate)))
	}
	return peakStarts
}

func computeFirstWordPeakStarts(samples []float64, frameRate int, groupedSplits, wordSplits []float64, duration float64) []float64 {
	if len(wordSplits) == 0 {
		return computePeakStarts(samples, frameRate, groupedSplits, duration)
	}

	peakStarts := make([]float64, 0, len(groupedSplits))
	for _, start := range groupedSplits {
		wordIndex := nearestSplitIndex(wordSplits, start)
		end := duration
		if wordIndex+1 < len(wordSplits) {
			end = wordSplits[wordIndex+1]
		}
		if end <= start {
			end = duration
		}
		peak := computePeakStarts(samples, frameRate, []float64{start, end}, duration)
		if len(peak) == 0 {
			peakStarts = append(peakStarts, round4(start))
			continue
		}
		peakStarts = append(peakStarts, peak[0])
	}
	return peakStarts
}

func nearestSplitIndex(splits []float64, value float64) int {
	best := 0
	bestDistance := math.Abs(splits[0] - value)
	for i := 1; i < len(splits); i++ {
		distance := math.Abs(splits[i] - value)
		if distance < bestDistance {
			best = i
			bestDistance = distance
		}
	}
	return best
}

func groupWordSplits(splits []float64, wordsPerSplit int) []float64 {
	if wordsPerSplit <= 1 || len(splits) <= 1 {
		return append([]float64(nil), splits...)
	}
	grouped := make([]float64, 0, (len(splits)+wordsPerSplit-1)/wordsPerSplit)
	for i := 0; i < len(splits); i += wordsPerSplit {
		grouped = append(grouped, splits[i])
	}
	if len(grouped) == 0 || grouped[0] != 0 {
		grouped = append([]float64{0}, grouped...)
	}
	return grouped
}

func detectSplitsSilence(samples []float64, frameRate int, duration, silenceThreshold, minSilenceSec float64) []float64 {
	blockSec := 0.05
	blockSize := int(float64(frameRate) * blockSec)
	minBlocks := max(1, int(minSilenceSec/blockSec))
	numBlocks := len(samples) / blockSize

	silent := make([]bool, numBlocks)
	for i := 0; i < numBlocks; i++ {
		chunk := samples[i*blockSize : (i+1)*blockSize]
		silent[i] = rms(chunk) < silenceThreshold
	}

	splits := []float64{0}
	for i := 0; i < len(silent); {
		if !silent[i] {
			i++
			continue
		}
		runStart := i
		for i < len(silent) && silent[i] {
			i++
		}
		runEnd := i
		if runEnd-runStart >= minBlocks {
			midT := float64((runStart+runEnd)/2) * blockSec
			if midT > 0 {
				splits = append(splits, round4(midT))
			}
		}
	}
	return splits
}

func detectSplitsFixed(duration, intervalSec float64) []float64 {
	var splits []float64
	for t := 0.0; t < duration; t += intervalSec {
		splits = append(splits, round4(t))
	}
	return splits
}

func detectSplitsWords(samples []float64, frameRate int, duration, silenceThreshold, minSilenceSec, minWordSec, maxWordSec float64) []float64 {
	blockSec := 0.01
	blockSize := max(1, int(float64(frameRate)*blockSec))
	minGapBlocks := max(1, int(minSilenceSec/blockSec))
	minWordBlocks := max(1, int(minWordSec/blockSec))
	maxWordBlocks := max(minWordBlocks+1, int(maxWordSec/blockSec))
	numBlocks := len(samples) / blockSize
	if numBlocks == 0 {
		return []float64{0}
	}

	rmsValues := make([]float64, numBlocks)
	for i := 0; i < numBlocks; i++ {
		rmsValues[i] = rms(samples[i*blockSize : (i+1)*blockSize])
	}

	envelope := make([]float64, len(rmsValues))
	for i := range rmsValues {
		start := max(0, i-3)
		end := min(len(rmsValues), i+4)
		var sum float64
		for _, value := range rmsValues[start:end] {
			sum += value
		}
		envelope[i] = sum / float64(end-start)
	}

	sortedRMS := append([]float64(nil), rmsValues...)
	sort.Float64s(sortedRMS)
	noiseFloor := sortedRMS[max(0, int(float64(len(sortedRMS))*0.2)-1)]
	peak := 0.0
	for _, value := range envelope {
		peak = max(peak, value)
	}
	if peak == 0 {
		peak = 1
	}
	threshold := max(silenceThreshold, max(noiseFloor*3.0, peak*0.04))

	voiced := make([]bool, len(envelope))
	for i, value := range envelope {
		voiced[i] = value >= threshold
	}

	var runs [][2]int
	for i := 0; i < len(voiced); {
		if !voiced[i] {
			i++
			continue
		}
		start := i
		for i < len(voiced) && voiced[i] {
			i++
		}
		runs = append(runs, [2]int{start, i})
	}
	if len(runs) == 0 {
		return []float64{0}
	}

	merged := [][2]int{runs[0]}
	for _, run := range runs[1:] {
		last := &merged[len(merged)-1]
		if run[0]-last[1] < minGapBlocks {
			last[1] = run[1]
		} else {
			merged = append(merged, run)
		}
	}

	var splitBlocks []int
	for _, run := range merged {
		start, end := run[0], run[1]
		if end-start < minWordBlocks {
			continue
		}
		splitBlocks = append(splitBlocks, start)
		segmentStart := start
		for segmentStart+maxWordBlocks < end {
			searchStart := segmentStart + minWordBlocks
			searchEnd := min(segmentStart+maxWordBlocks, end-minWordBlocks)
			if searchEnd <= searchStart {
				break
			}
			valley := searchStart
			for idx := searchStart + 1; idx < searchEnd; idx++ {
				if envelope[idx] < envelope[valley] {
					valley = idx
				}
			}
			localPeak := 0.0
			for _, value := range envelope[segmentStart:searchEnd] {
				localPeak = max(localPeak, value)
			}
			if localPeak == 0 {
				localPeak = peak
			}
			if envelope[valley] < localPeak*wordValleySplitRatio {
				splitBlocks = append(splitBlocks, valley)
				segmentStart = valley
			} else {
				segmentStart += maxWordBlocks
			}
		}
	}
	if len(splitBlocks) == 0 {
		return []float64{0}
	}

	sort.Ints(splitBlocks)
	splits := make([]float64, 0, len(splitBlocks))
	lastBlock := -1
	for _, block := range splitBlocks {
		if block == lastBlock {
			continue
		}
		lastBlock = block
		t := round4(float64(block) * blockSec)
		if len(splits) == 0 || t-splits[len(splits)-1] >= minWordSec {
			splits = append(splits, t)
		}
	}

	splits[0] = 0
	return splits
}

func rms(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, sample := range samples {
		sum += sample * sample
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func maxWindowRMS(samples []float64, frameRate int, windowSeconds float64) float64 {
	if len(samples) == 0 || frameRate <= 0 {
		return 0
	}
	windowSize := max(1, int(float64(frameRate)*windowSeconds))
	maximum := 0.0
	for start := 0; start < len(samples); start += windowSize {
		end := min(len(samples), start+windowSize)
		maximum = max(maximum, rms(samples[start:end]))
	}
	return maximum
}

func exceedsWordThreshold(maxRMS, threshold float64) bool {
	threshold = clampWordThreshold(threshold)
	return threshold <= 0 || maxRMS > threshold
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
