#!/bin/bash
# go-polyscript Benchmark Runner
#
# This script runs benchmarks for go-polyscript and generates comparison reports.
# It automatically saves results to the benchmark_results/ directory and compares
# with previous runs to track performance changes over time.
#
# Usage:
#   ./run.sh                          # Run all benchmarks
#   ./run.sh <pattern>                # Run benchmarks matching pattern
#   ./run.sh <pattern> <iterations>   # Run with specific number of iterations (default: auto)
#
# Examples:
#   ./run.sh                           # Run all benchmarks
#   ./run.sh BenchmarkEvaluationPatterns  # Run just evaluation pattern benchmarks
#   ./run.sh CompositeProvider 20x     # Run CompositeProvider benchmarks with 20 iterations
#   ./run.sh RisorVM 5x                # Run RisorVM benchmarks with just 5 iterations
#
# The script saves results to benchmark/results/ directory and creates comparison reports
# to help identify performance improvements or regressions between changes.

set -e

# Find the repo root directory (parent of the benchmarks directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to the repo root for running tests
cd "$REPO_ROOT"

# Print banner
echo "======================================================"
echo "   go-polyscript Benchmarks"
echo "======================================================"
echo ""

# Create benchmark results directory if it doesn't exist
RESULTS_DIR="$SCRIPT_DIR/results"
mkdir -p $RESULTS_DIR

# Get current date for results filename
DATE=$(date +"%Y-%m-%d_%H-%M-%S")
RESULTS_FILE="$RESULTS_DIR/benchmark_$DATE.txt"
RESULTS_JSON="$RESULTS_DIR/benchmark_$DATE.json"
LATEST_LINK="$RESULTS_DIR/latest.txt"
COMPARE_FILE="$RESULTS_DIR/comparison.txt"

# Check if arguments were provided
PATTERN="."
BENCHTIME=""

if [ -n "$1" ]; then
    PATTERN="$1"
    echo "Running benchmarks matching pattern: $PATTERN"
else
    echo "Running all benchmarks"
fi

if [ -n "$2" ]; then
    BENCHTIME="-benchtime=$2"
    echo "Running with custom iterations: $2"
fi

# Use a more friendly path for display in output
FRIENDLY_RESULTS_PATH="results/benchmark_$DATE.txt"
echo "Results will be saved to $FRIENDLY_RESULTS_PATH"
echo ""

# Run benchmarks and save results (from the repo root)
go test -bench=$PATTERN $BENCHTIME -benchmem ./engine -run=^$ | tee $RESULTS_FILE

# Also save JSON format for programmatic analysis
go test -bench=$PATTERN $BENCHTIME -benchmem ./engine -run=^$ -json > $RESULTS_JSON

# Update "latest" symlink
ln -sf "benchmark_$DATE.txt" $LATEST_LINK

# Check if we have previous results to compare with
PREV_RESULTS=$(find $RESULTS_DIR -name "benchmark_*.txt" -not -name "benchmark_$DATE.txt" | sort -r | head -n 1)

if [ -n "$PREV_RESULTS" ]; then
    PREV_BASENAME=$(basename $PREV_RESULTS)
    CURR_BASENAME=$(basename $RESULTS_FILE)
    
    echo ""
    echo "Comparing with previous results: $PREV_BASENAME"
    echo "======================================================"
    
    # Use benchstat for comparison if available
    if command -v benchstat &> /dev/null; then
        # Use custom file labels to show shorter paths
        # This uses the benchstat's label=path feature mentioned in the docs
        benchstat "previous=$PREV_RESULTS" "current=$RESULTS_FILE" | tee $COMPARE_FILE
    else
        echo "benchstat not found. Install with: go install golang.org/x/perf/cmd/benchstat@latest"
        echo "Manual comparison:"
        echo ""
        echo "Previous results:"
        cat $PREV_RESULTS
        echo ""
        echo "Current results:"
        cat $RESULTS_FILE
    fi
fi

echo ""
echo "Benchmarks complete. Results saved to $FRIENDLY_RESULTS_PATH"
echo "======================================================"