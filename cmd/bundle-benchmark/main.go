package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"time"

	"github.com/klauspost/compress/zstd"

	"gitlab.com/mainops/uniShell/internal/bundle"
)

const (
	warmupRuns   = 5
	measuredRuns = 10
)

var zstdLevels = []struct {
	name  string
	level zstd.EncoderLevel
}{
	{
		name:  "fastest",
		level: zstd.SpeedFastest,
	},
	{
		name:  "default",
		level: zstd.SpeedDefault,
	},
	{
		name:  "better",
		level: zstd.SpeedBetterCompression,
	},
	{
		name:  "best",
		level: zstd.SpeedBestCompression,
	},
}

type result struct {
	Algorithm           string  `json:"algorithm"`
	Level               string  `json:"level"`
	ArchiveBytes        int     `json:"archive_bytes"`
	CompressedBytes     int     `json:"compressed_bytes"`
	ReductionPercent    float64 `json:"reduction_percent"`
	CompressionMin      float64 `json:"compression_min_seconds"`
	CompressionMedian   float64 `json:"compression_median_seconds"`
	CompressionMean     float64 `json:"compression_mean_seconds"`
	CompressionMax      float64 `json:"compression_max_seconds"`
	DecompressionMin    float64 `json:"decompression_min_seconds"`
	DecompressionMedian float64 `json:"decompression_median_seconds"`
	DecompressionMean   float64 `json:"decompression_mean_seconds"`
	DecompressionMax    float64 `json:"decompression_max_seconds"`
	CompressionAllocs   uint64  `json:"compression_allocs"`
	DecompressionAllocs uint64  `json:"decompression_allocs"`
}

type benchmarkReport struct {
	Commit    string   `json:"commit"`
	GoVersion string   `json:"go_version"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	Warmup    int      `json:"warmup_runs"`
	Measured  int      `json:"measured_runs"`
	Results   []result `json:"results"`
}

type compressor interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	Close() error
}

type zstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func newZstdCompressor(level zstd.EncoderLevel) (*zstdCompressor, error) {
	encoder, err := zstd.NewWriter(
		nil,
		zstd.WithEncoderLevel(level),
	)
	if err != nil {
		return nil, err
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	return &zstdCompressor{
		encoder: encoder,
		decoder: decoder,
	}, nil
}

func (c *zstdCompressor) Compress(data []byte) ([]byte, error) {
	return c.encoder.EncodeAll(data, nil), nil
}

func (c *zstdCompressor) Decompress(data []byte) ([]byte, error) {
	return c.decoder.DecodeAll(data, nil)
}

func (c *zstdCompressor) Close() error {
	return c.encoder.Close()
}

type gzipCompressor struct{}

func (gzipCompressor) Compress(data []byte) ([]byte, error) {
	var output bytes.Buffer

	writer, err := gzip.NewWriterLevel(
		&output,
		gzip.DefaultCompression,
	)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func (gzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var output bytes.Buffer

	if _, err := output.ReadFrom(reader); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func (gzipCompressor) Close() error {
	return nil
}

func main() {
	input := flag.String(
		"input",
		"assets",
		"directory containing runtime assets",
	)
	output := flag.String(
		"output",
		"",
		"optional JSON report path",
	)
	commit := flag.String(
		"commit",
		"",
		"repository commit identifier",
	)
	flag.Parse()

	if err := run(*input, *output, *commit); err != nil {
		fmt.Fprintf(os.Stderr, "bundle-benchmark: %v\n", err)
		os.Exit(1)
	}
}

func run(input, output, commit string) error {
	archive, err := bundle.CreateArchive(input)
	if err != nil {
		return fmt.Errorf("create runtime archive: %w", err)
	}

	report := benchmarkReport{
		Commit:    commit,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Warmup:    warmupRuns,
		Measured:  measuredRuns,
	}

	for _, candidate := range zstdLevels {
		compressor, err := newZstdCompressor(candidate.level)
		if err != nil {
			return fmt.Errorf(
				"create zstd compressor %s: %w",
				candidate.name,
				err,
			)
		}

		item, err := benchmark(
			"zstd",
			candidate.name,
			compressor,
			archive,
		)

		closeErr := compressor.Close()

		if err != nil {
			return fmt.Errorf(
				"benchmark zstd %s: %w",
				candidate.name,
				err,
			)
		}

		if closeErr != nil {
			return fmt.Errorf(
				"close zstd %s: %w",
				candidate.name,
				closeErr,
			)
		}

		report.Results = append(report.Results, item)
	}

	sort.Slice(
		report.Results,
		func(i, j int) bool {
			order := map[string]int{
				"fastest": 0,
				"default": 1,
				"better":  2,
				"best":    3,
			}

			return order[report.Results[i].Level] <
				order[report.Results[j].Level]
		},
	)

	printReport(report)

	if output != "" {
		if err := writeReport(output, report); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	return nil
}

func benchmark(
	algorithm string,
	level string,
	compressor compressor,
	archive []byte,
) (result, error) {
	var compressed []byte

	for range warmupRuns {
		var err error

		compressed, err = compressor.Compress(archive)
		if err != nil {
			return result{}, err
		}
	}

	compressed, compressionTimes, compressionAllocs, err :=
		measureCompression(compressor, archive)

	if err != nil {
		return result{}, err
	}

	for range warmupRuns {
		decompressed, err := compressor.Decompress(compressed)
		if err != nil {
			return result{}, err
		}

		if !bytes.Equal(decompressed, archive) {
			return result{}, fmt.Errorf(
				"warm-up decompressed archive does not match original",
			)
		}
	}

	decompressed, decompressionTimes, decompressionAllocs, err :=
		measureDecompression(compressor, compressed)

	if err != nil {
		return result{}, err
	}

	if !bytes.Equal(decompressed, archive) {
		return result{}, fmt.Errorf(
			"decompressed archive does not match original",
		)
	}

	reduction := 0.0
	if len(archive) > 0 {
		reduction =
			(1 -
				float64(len(compressed))/float64(len(archive))) *
				100
	}

	return result{
		Algorithm:           algorithm,
		Level:               level,
		ArchiveBytes:        len(archive),
		CompressedBytes:     len(compressed),
		ReductionPercent:    reduction,
		CompressionMin:      durationMin(compressionTimes),
		CompressionMedian:   durationMedian(compressionTimes),
		CompressionMean:     durationMean(compressionTimes),
		CompressionMax:      durationMax(compressionTimes),
		DecompressionMin:    durationMin(decompressionTimes),
		DecompressionMedian: durationMedian(decompressionTimes),
		DecompressionMean:   durationMean(decompressionTimes),
		DecompressionMax:    durationMax(decompressionTimes),
		CompressionAllocs:   compressionAllocs,
		DecompressionAllocs: decompressionAllocs,
	}, nil
}

func measureCompression(
	compressor compressor,
	archive []byte,
) ([]byte, []float64, uint64, error) {
	var compressed []byte
	times := make([]float64, 0, measuredRuns)

	var totalAllocs uint64

	for range measuredRuns {
		runtime.GC()
		debug.FreeOSMemory()

		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		start := time.Now()

		result, err := compressor.Compress(archive)

		elapsed := time.Since(start)

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		if err != nil {
			return nil, nil, 0, err
		}

		compressed = result
		times = append(times, elapsed.Seconds())
		totalAllocs += after.Mallocs - before.Mallocs
	}

	return compressed, times, totalAllocs / measuredRuns, nil
}

func measureDecompression(
	compressor compressor,
	compressed []byte,
) ([]byte, []float64, uint64, error) {
	var decompressed []byte
	times := make([]float64, 0, measuredRuns)

	var totalAllocs uint64

	for range measuredRuns {
		runtime.GC()
		debug.FreeOSMemory()

		var before runtime.MemStats
		runtime.ReadMemStats(&before)

		start := time.Now()

		result, err := compressor.Decompress(compressed)

		elapsed := time.Since(start)

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		if err != nil {
			return nil, nil, 0, err
		}

		decompressed = result
		times = append(times, elapsed.Seconds())
		totalAllocs += after.Mallocs - before.Mallocs
	}

	return decompressed, times, totalAllocs / measuredRuns, nil
}

func durationMin(values []float64) float64 {
	result := values[0]

	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}

	return result
}

func durationMedian(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	middle := len(sorted) / 2

	if len(sorted)%2 == 0 {
		return (sorted[middle-1] + sorted[middle]) / 2
	}

	return sorted[middle]
}

func durationMean(values []float64) float64 {
	var total float64

	for _, value := range values {
		total += value
	}

	return total / float64(len(values))
}

func durationMax(values []float64) float64 {
	result := values[0]

	for _, value := range values[1:] {
		if value > result {
			result = value
		}
	}

	return result
}

func printReport(report benchmarkReport) {
	fmt.Println("uniShell bundle compression benchmark")
	fmt.Println()

	fmt.Printf("Go:       %s\n", report.GoVersion)
	fmt.Printf("OS:       %s\n", report.OS)
	fmt.Printf("Arch:     %s\n", report.Arch)
	fmt.Printf("Commit:   %s\n", report.Commit)
	fmt.Printf("Warm-up:  %d runs\n", report.Warmup)
	fmt.Printf("Measured: %d runs\n", report.Measured)
	fmt.Println()

	if len(report.Results) == 0 {
		return
	}

	fmt.Printf(
		"Archive:  %d bytes\n",
		report.Results[0].ArchiveBytes,
	)
	fmt.Println()

	fmt.Printf(
		"%-8s %12s %14s %12s %12s %12s %12s\n",
		"Level",
		"Compressed",
		"Reduction",
		"Comp med",
		"Comp mean",
		"Decomp med",
		"Decomp mean",
	)

	fmt.Println(
		"--------------------------------------------------------------------------------",
	)

	for _, item := range report.Results {
		fmt.Printf(
			"%-8s %14d %11.2f%% %11.6fs %11.6fs %13.6fs %13.6fs\n",
			item.Level,
			item.CompressedBytes,
			item.ReductionPercent,
			item.CompressionMedian,
			item.CompressionMean,
			item.DecompressionMedian,
			item.DecompressionMean,
		)
	}

	fmt.Println()

	smallest := report.Results[0]

	for _, item := range report.Results[1:] {
		if item.CompressedBytes < smallest.CompressedBytes {
			smallest = item
		}
	}

	fmt.Printf(
		"Smallest:  %s (%d bytes)\n",
		smallest.Level,
		smallest.CompressedBytes,
	)

	fastestDecompression := report.Results[0]

	for _, item := range report.Results[1:] {
		if item.DecompressionMedian <
			fastestDecompression.DecompressionMedian {
			fastestDecompression = item
		}
	}

	fmt.Printf(
		"Fastest decompression: %s (%.6fs median)\n",
		fastestDecompression.Level,
		fastestDecompression.DecompressionMedian,
	)

	fastestCompression := report.Results[0]

	for _, item := range report.Results[1:] {
		if item.CompressionMedian <
			fastestCompression.CompressionMedian {
			fastestCompression = item
		}
	}

	fmt.Printf(
		"Fastest compression: %s (%.6fs median)\n",
		fastestCompression.Level,
		fastestCompression.CompressionMedian,
	)
}

func writeReport(path string, report benchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	return os.WriteFile(path, data, 0600)
}
