package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/apache/beam/sdks/go/pkg/beam"
	"github.com/apache/beam/sdks/go/pkg/beam/io/textio"
	_ "github.com/apache/beam/sdks/go/pkg/beam/runners/spark"
	"github.com/apache/beam/sdks/go/pkg/beam/runners/universal"
	"github.com/apache/beam/sdks/go/pkg/beam/transforms/stats"
	"github.com/apache/beam/sdks/go/pkg/beam/x/beamx"
)

// Execute runs the given pipeline on Spark. Convenience wrapper over the
// universal runner.
func Execute(ctx context.Context, p *beam.Pipeline) error {
	return universal.Execute(ctx, p)
}

// Concept #2: Defining your own configuration options. Pipeline options can
// just be standard Go flags (or be obtained any other way). Defining and
// configuring the pipeline is normal Go code.
var (
	// By default, this example reads from a public dataset containing the text of
	// King Lear. Set this option to choose a different input file or glob.
	input = flag.String("input", "gs://apache-beam-samples/shakespeare/kinglear.txt", "File(s) to read.")

	// Set this required option to specify where to write the output.
	output = flag.String("output", "", "Output file (required).")
)

// Concept #3: You can make your pipeline assembly code less verbose and by
// defining your DoFns statically out-of-line. A DoFn can be defined as a Go
// function and is conventionally suffixed "Fn". The argument and return types
// dictate the pipeline shape when used in a ParDo: for example,
//
//      formatFn: string x int -> string
//
// indicate that it operates on a PCollection of type KV<string,int>, representing
// key value pairs of strings and ints, and outputs a PCollection of type string.
// Beam typechecks the pipeline before running it.
//
// DoFns that potentially output zero or multiple elements can also be Go functions,
// but have a different signature. For example,
//
//       extractFn : string x func(string) -> ()
//
// uses an "emit" function argument instead of string return type to allow it to
// output any number of elements. It operates on a PCollection of type string and
// returns a PCollection of type string. Also, using named function transforms allows
// for easy reuse, modular testing, and an improved monitoring experience.

var (
	wordRE  = regexp.MustCompile(`[a-zA-Z]+('[a-z])?`)
	empty   = beam.NewCounter("extract", "emptyLines")
	lineLen = beam.NewDistribution("extract", "lineLenDistro")
)

// extractFn is a DoFn that emits the words in a given line.
func extractFn(ctx context.Context, line string, emit func(string)) {
	lineLen.Update(ctx, int64(len(line)))
	if len(strings.TrimSpace(line)) == 0 {
		empty.Inc(ctx, 1)
	}
	for _, word := range wordRE.FindAllString(line, -1) {
		emit(word)
	}
}

// formatFn is a DoFn that formats a word and its count as a string.
func formatFn(w string, c int) string {
	return fmt.Sprintf("%s: %v", w, c)
}

// Concept #4: A composite PTransform is a Go function that adds
// transformations to a given pipeline. It is run at construction time and
// works on PCollections as values. For monitoring purposes, the pipeline
// allows scoped naming for composite transforms. The difference between a
// composite transform and a construction helper function is solely in whether
// a scoped name is used.
//
// For example, the CountWords function is a custom composite transform that
// bundles two transforms (ParDo and Count) as a reusable function.

// CountWords is a composite transform that counts the words of a PCollection
// of lines. It expects a PCollection of type string and returns a PCollection
// of type KV<string,int>. The Beam type checker enforces these constraints
// during pipeline construction.
func CountWords(s beam.Scope, lines beam.PCollection) beam.PCollection {
	s = s.Scope("CountWords")

	// Convert lines of text into individual words.
	col := beam.ParDo(s, extractFn, lines)

	// Count the number of times each word occurs.
	return stats.Count(s, col)
}

func main() {
	// If beamx or Go flags are used, flags must be parsed first.
	flag.Parse()
	// beam.Init() is an initialization hook that must be called on startup. On
	// distributed runners, it is used to intercept control.
	beam.Init()
	//beam.RegisterRunner("spark", Execute)

	// Input validation is done as usual. Note that it must be after Init().
	if *output == "" {
		log.Fatal("No output provided")
	}

	// Concepts #3 and #4: The pipeline uses the named transform and DoFn.
	p := beam.NewPipeline()
	s := p.Root()

	lines := textio.Read(s, *input)
	counted := CountWords(s, lines)
	formatted := beam.ParDo(s, formatFn, counted)
	textio.Write(s, *output, formatted)

	// Concept #1: The beamx.Run convenience wrapper allows a number of
	// pre-defined runners to be used via the --runner flag.
	if err := beamx.Run(context.Background(), p); err != nil {
		log.Fatalf("Failed to execute job: %v", err)
	}
}
