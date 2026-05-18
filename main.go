package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const GsubVersion = "1.1.0"

// Global compiled regex optimized for reuse across lines and streams
var placeholderRegex = regexp.MustCompile(`(\\)?\{\{\s*((?:\$[a-zA-Z0-9]+\.)?[a-zA-Z0-9._-]+)\s*(?:\|\|\s*'([^']*)')?\s*\}\}`)

func main() {
	// 1. Setup Strict CLI Flags
	useEnv := flag.Bool("env", false, "Source from system environment")
	flag.BoolVar(useEnv, "e", false, "Source from system environment (shorthand)")

	envFilePath := flag.String("file", "", "Path to a .env file (use '-' to read variables from STDIN)")
	flag.StringVar(envFilePath, "f", "", "Path to a .env file (use '-' to read variables from STDIN) (shorthand)")

	templatePath := flag.String("template", "", "Path to the template file (if omitted, reads template structure from STDIN)")
	flag.StringVar(templatePath, "t", "", "Path to the template file (shorthand)")

	version := flag.Bool("version", false, "Show version")
	flag.BoolVar(version, "v", false, "Show version (shorthand)")

	prefixPassed := flag.Bool("prefix", false, "Enable stderr prefixing (omitted: no prefix, -p only: '[gsub] ', -p value: 'value ')")
	flag.BoolVar(prefixPassed, "p", false, "Enable stderr prefixing (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gsub [OPTION]...\n")
		fmt.Fprintf(os.Stderr, "Substitute {{PLACEHOLDERS}} safely using environment variables or configurations.\n\n")
		fmt.Fprintf(os.Stderr, "Example: echo \"Hello {{USER}}\" | gsub -e\n")
		fmt.Fprintf(os.Stderr, "         echo -e \"USER=alpine\nTIMESTAMP=$(date +%%s)\" | gsub -t config.json.tmpl -f -\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", GsubVersion)
		return
	}

	// Short Usage Protection Guard
	stat, _ := os.Stdin.Stat()
	isTerminal := (stat.Mode() & os.ModeCharDevice) != 0

	if isTerminal && flag.NFlag() == 0 && *templatePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: gsub [OPTION]...\nTry 'gsub --help' for more information.\n")
		os.Exit(1)
	}

	// 2. Resolve Stderr Logging Prefix
	var activePrefix string
	if *prefixPassed {
		activePrefix = "[gsub] "

		args := os.Args[1:]
		for i, arg := range args {
			if arg == "-p" || arg == "--prefix" || strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--prefix=") {
				var potentialValue string

				if strings.Contains(arg, "=") {
					potentialValue = strings.SplitN(arg, "=", 2)[1]
				} else if i+1 < len(args) {
					potentialValue = args[i+1]
				}

				if potentialValue != "" && !strings.HasPrefix(potentialValue, "-") {
					activePrefix = potentialValue + " "
				}
				break
			}
		}
	}

	// 3. Runtime Strategy Forking
	if *templatePath != "" && *envFilePath == "-" {
		// MODE 2: Continuous Telemetry Streaming Engine
		// Completely ingest the template metadata file into RAM once to free STDIN for data frames
		templateLines, err := readTemplateLines(*templatePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sstate=failed reason=template_missing target=\"%s\"\n", activePrefix, *templatePath)
			os.Exit(1)
		}
		runContinuousStream(templateLines, *useEnv, activePrefix)
	} else {
		// MODE 1: Static Configuration One-Shot Mode
		fileVars := make(map[string]string)

		// Layer A: Load static variables from physical file if provided
		if *envFilePath != "" && *envFilePath != "-" {
			file, err := os.Open(*envFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=file_missing target=\"%s\"\n", activePrefix, *envFilePath)
				os.Exit(1)
			}
			vars, err := parseEnvReader(file)
			file.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=parse_error message=\"file: %v\"\n", activePrefix, err)
				os.Exit(1)
			}
			fileVars = vars
		}

		// Layer B: Ingest STDIN as dynamic variables if a template file path is explicitly targeted
		if *templatePath != "" {
			// Ensure STDIN is actually hooked to a pipe or file redirection to protect against terminal lockups
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				stdinVars, err := parseEnvReader(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%sstate=failed reason=parse_error message=\"stdin: %v\"\n", activePrefix, err)
					os.Exit(1)
				}
				// Merge inputs; standard input key definitions take precedence over underlying configuration files
				for k, v := range stdinVars {
					fileVars[k] = v
				}
			}
		}

		// Layer C: Bind target template parsing layout stream
		var templateReader io.Reader
		if *templatePath != "" {
			file, err := os.Open(*templatePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=template_missing target=\"%s\"\n", activePrefix, *templatePath)
				os.Exit(1)
			}
			defer file.Close()
			templateReader = file
		} else {
			if *envFilePath == "-" {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=deadlock message=\"cannot read env variables and template data from STDIN simultaneously\"\n", activePrefix)
				os.Exit(1)
			}
			templateReader = os.Stdin
		}

		executeOneShot(templateReader, fileVars, *useEnv, activePrefix)
	}
}

// readTemplateLines pre-loads template arrays into memory for non-blocking stream compilation
func readTemplateLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// processLine parses strings via continuous token analysis with strict presence evaluation rules
func processLine(line string, fileVars map[string]string, useEnv bool, missingVariables map[string]bool) string {
	return placeholderRegex.ReplaceAllStringFunc(line, func(match string) string {
		submatches := placeholderRegex.FindStringSubmatch(match)

		// Transparent Escaping Architecture Evaluation (Group index 1 captures backslash matches)
		if submatches[1] != "" {
			return match[1:] // Safely strip the backslash and output raw placeholder syntax unchanged
		}

		fullKey := submatches[2]
		hasDefault := submatches[3] != "" || strings.Contains(match, "||")
		defaultValue := submatches[3]

			var val string
			var exists bool

			switch {
				case strings.HasPrefix(fullKey, "$env."):
					realKey := strings.TrimPrefix(fullKey, "$env.")
					val, exists = os.LookupEnv(realKey)

				case strings.HasPrefix(fullKey, "$file.") || strings.HasPrefix(fullKey, "$input."):
					realKey := fullKey[strings.Index(fullKey, ".")+1:]
					val, exists = fileVars[realKey]

				default:
					// Un-namespaced Cascade Logic: Environment takes priority if explicit flag is provided
					if useEnv {
						val, exists = os.LookupEnv(fullKey)
					}
					if !exists {
						val, exists = fileVars[fullKey]
					}
			}

			// Hard Existence Safeguard: Empty strings pass cleanly, non-existent entries trigger defaults or failures
			if !exists {
				if hasDefault {
					return defaultValue
				}
				missingVariables[fullKey] = true
				return match
			}
			return val
	})
}

// executeOneShot maps static template evaluation workflows sequentially (Mode 1)
func executeOneShot(r io.Reader, fileVars map[string]string, useEnv bool, prefix string) {
	var outBuffer bytes.Buffer
	missingVariables := make(map[string]bool)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		outBuffer.WriteString(processLine(scanner.Text(), fileVars, useEnv, missingVariables) + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%sstate=failed reason=stream_error message=\"%v\"\n", prefix, err)
		os.Exit(1)
	}

	if len(missingVariables) > 0 {
		failLoud(missingVariables, prefix)
	}
	_, _ = outBuffer.WriteTo(os.Stdout)
}

// runContinuousStream scales key value maps indefinitely over persistent telemetry feeds (Mode 2)
func runContinuousStream(templateLines []string, useEnv bool, prefix string) {
	scanner := bufio.NewScanner(os.Stdin)
	currentFrameVars := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Frame Boundary Trigger: Empty newline triggers instant-flush of compiled blocks downstream
		if line == "" {
			if len(currentFrameVars) > 0 {
				flushFrame(templateLines, currentFrameVars, useEnv, prefix)
				currentFrameVars = make(map[string]string) // Clean context maps instantly for next tick
			}
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			currentFrameVars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Catch trailing frame data if input streams terminate with open values
	if len(currentFrameVars) > 0 {
		flushFrame(templateLines, currentFrameVars, useEnv, prefix)
	}
}

// flushFrame evaluates structured metrics contexts directly to stdout without dropping daemon loop allocations
func flushFrame(lines []string, vars map[string]string, useEnv bool, prefix string) {
	var frameBuffer bytes.Buffer
	missingVariables := make(map[string]bool)

	for _, line := range lines {
		frameBuffer.WriteString(processLine(line, vars, useEnv, missingVariables) + "\n")
	}

	if len(missingVariables) > 0 {
		failLoud(missingVariables, prefix) // Pipeline terminates loud and fast on infrastructure variance
	}

	_, _ = frameBuffer.WriteTo(os.Stdout)
}

// failLoud structures standardized diagnostics directly to STDERR for unified cloud logger mapping
func failLoud(missingVars map[string]bool, prefix string) {
	keys := make([]string, 0, len(missingVars))
	for k := range missingVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Fprintf(os.Stderr, "%sstate=failed reason=missing_vars targets=%s\n", prefix, strings.Join(keys, ","))
	os.Exit(1)
}

// parseEnvReader consumes localized streams into flat data lookups
func parseEnvReader(r io.Reader) (map[string]string, error) {
	vars := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return vars, scanner.Err()
}
