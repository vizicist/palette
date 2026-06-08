package samplesplitter

import (
	"errors"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const minimumMP3DurationSeconds = 10.0

type MP3File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ListMP3Files(dir string) ([]MP3File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]MP3File, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".mp3" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		duration, err := MP3DurationSeconds(path)
		if err != nil || duration < minimumMP3DurationSeconds {
			continue
		}
		files = append(files, MP3File{
			Name: entry.Name(),
			Path: path,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return files, nil
}

func ChooseRandomPrefixedMP3(dir, prefix string, rng *rand.Rand) (MP3File, error) {
	return ChooseRandomPrefixedMP3Excluding(dir, prefix, "", rng)
}

func ChooseRandomPrefixedMP3Excluding(dir, prefix, excludePath string, rng *rand.Rand) (MP3File, error) {
	files, err := ListMP3Files(dir)
	if err != nil {
		return MP3File{}, err
	}
	matches := make([]MP3File, 0)
	alternates := make([]MP3File, 0)
	excludePath = normalizePathForCompare(excludePath)
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name), strings.ToLower(prefix)) {
			matches = append(matches, file)
			if normalizePathForCompare(file.Path) != excludePath {
				alternates = append(alternates, file)
			}
		}
	}
	if len(matches) == 0 {
		return MP3File{}, fs.ErrNotExist
	}
	if len(alternates) > 0 {
		matches = alternates
	}
	if rng == nil {
		return matches[rand.Intn(len(matches))], nil
	}
	return matches[rng.Intn(len(matches))], nil
}

func normalizePathForCompare(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return strings.ToLower(filepath.Clean(abs))
}

func ResolveMP3File(dir, filename string) (string, error) {
	if filename == "" {
		return "", errors.New("missing file")
	}

	base, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(base, filename))
	if err != nil {
		return "", err
	}
	if filepath.Dir(candidate) != base || strings.ToLower(filepath.Ext(candidate)) != ".mp3" {
		return "", errors.New("file must be an MP3 directly inside the configured directory")
	}
	if _, err := os.Stat(candidate); err != nil {
		return "", err
	}
	duration, err := MP3DurationSeconds(candidate)
	if err != nil {
		return "", err
	}
	if duration < minimumMP3DurationSeconds {
		return "", errors.New("MP3 must be at least 10 seconds long")
	}
	return candidate, nil
}

func MP3DurationSeconds(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	offset := skipID3v2(data)
	duration := 0.0
	frames := 0
	for offset+4 <= len(data) {
		frame, ok := parseMP3FrameHeader(data[offset : offset+4])
		if !ok {
			offset++
			continue
		}
		if offset+frame.length > len(data) {
			break
		}
		duration += float64(frame.samples) / float64(frame.sampleRate)
		frames++
		offset += frame.length
	}
	if frames == 0 {
		return 0, errors.New("no MP3 frames found")
	}
	return duration, nil
}

func skipID3v2(data []byte) int {
	if len(data) < 10 || string(data[:3]) != "ID3" {
		return 0
	}
	size := int(data[6]&0x7f)<<21 | int(data[7]&0x7f)<<14 | int(data[8]&0x7f)<<7 | int(data[9]&0x7f)
	return min(len(data), 10+size)
}

type mp3Frame struct {
	length     int
	sampleRate int
	samples    int
}

func parseMP3FrameHeader(header []byte) (mp3Frame, bool) {
	if len(header) < 4 || header[0] != 0xff || header[1]&0xe0 != 0xe0 {
		return mp3Frame{}, false
	}
	versionBits := (header[1] >> 3) & 0x03
	layerBits := (header[1] >> 1) & 0x03
	bitrateIndex := (header[2] >> 4) & 0x0f
	sampleRateIndex := (header[2] >> 2) & 0x03
	padding := int((header[2] >> 1) & 0x01)
	if versionBits == 0x01 || layerBits == 0 || bitrateIndex == 0 || bitrateIndex == 0x0f || sampleRateIndex == 0x03 {
		return mp3Frame{}, false
	}

	bitrate := mp3BitrateKbps(versionBits, layerBits, bitrateIndex) * 1000
	sampleRate := mp3SampleRate(versionBits, sampleRateIndex)
	if bitrate == 0 || sampleRate == 0 {
		return mp3Frame{}, false
	}

	samples := mp3SamplesPerFrame(versionBits, layerBits)
	length := mp3FrameLength(versionBits, layerBits, bitrate, sampleRate, padding)
	if samples == 0 || length <= 4 {
		return mp3Frame{}, false
	}
	return mp3Frame{length: length, sampleRate: sampleRate, samples: samples}, true
}

func mp3BitrateKbps(versionBits, layerBits, index byte) int {
	table := map[byte]map[byte][]int{
		0x03: {
			0x03: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
			0x02: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
			0x01: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
		},
		0x02: {
			0x03: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
			0x02: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
			0x01: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		},
		0x00: {
			0x03: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
			0x02: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
			0x01: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		},
	}
	return table[versionBits][layerBits][index]
}

func mp3SampleRate(versionBits, index byte) int {
	rates := []int{44100, 48000, 32000}
	rate := rates[index]
	switch versionBits {
	case 0x02:
		return rate / 2
	case 0x00:
		return rate / 4
	default:
		return rate
	}
}

func mp3SamplesPerFrame(versionBits, layerBits byte) int {
	switch layerBits {
	case 0x03:
		return 384
	case 0x02:
		return 1152
	case 0x01:
		if versionBits == 0x03 {
			return 1152
		}
		return 576
	default:
		return 0
	}
}

func mp3FrameLength(versionBits, layerBits byte, bitrate, sampleRate, padding int) int {
	if layerBits == 0x03 {
		return (12*bitrate/sampleRate + padding) * 4
	}
	if layerBits == 0x01 && versionBits != 0x03 {
		return 72*bitrate/sampleRate + padding
	}
	return 144*bitrate/sampleRate + padding
}
