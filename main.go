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

const GsubVersion = "1.0.0"

func main() {
	// 1. Setup Flags
	useEnv := flag.Bool("env", false, "Source from system environment")
	flag.BoolVar(useEnv, "e", false, "Source from system environment (shorth)")

	envFilePath := flag.String("file", "", "Path to a .env file (use '-' to read from STDIN)")
	flag.StringVar(envFilePath, "f", "", "Path to a .env file (use '-' to read from STDIN) (shorth)")

	templatePath := flag.String("template", "", "Path to the template file (if omitted, reads template from STDIN)")
	flag.StringVar(templatePath, "t", "", "Path to the template file (shorth)")

	allowMissing := flag.Bool("allow-missing", false, "Allow placeholders to remain")
	flag.BoolVar(allowMissing, "a", false, "Allow placeholders to remain (shorth)")

	version := flag.Bool("version", false, "Show version")
	flag.BoolVar(version, "v", false, "Show version (shorth)")

	prefixPassed := flag.Bool("prefix", false, "Enable stderr prefixing (omitted: no prefix, -p only: '[gsub] ', -p value: 'value ')")
	flag.BoolVar(prefixPassed, "p", false, "Enable stderr prefixing (shorth)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gsub [OPTION]...\n")
		fmt.Fprintf(os.Stderr, "Substitute {{PLACEHOLDERS}} using environment variables or configurations.\n\n")
		fmt.Fprintf(os.Stderr, "Example: echo \"Hello {{USER}}\" | gsub -e\n")
		fmt.Fprintf(os.Stderr, "         gmon | gsub -t config.json.tmpl -f -\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExit status is 0 if substitution succeeds, 1 if variables are missing and -a is not used.\n")
	}

	flag.Parse()

	if *version {
		fmt.Printf("v%s\n", GsubVersion)
		return
	}

	// 2. Short Usage Logic
	stat, _ := os.Stdin.Stat()
	isTerminal := (stat.Mode() & os.ModeCharDevice) != 0

	if isTerminal && flag.NFlag() == 0 && *templatePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: gsub [OPTION]...\n")
		fmt.Fprintf(os.Stderr, "Try 'gsub --help' for more information.\n")
		os.Exit(1)
	}

	// 3. Handle Prefix Logic
	var activePrefix string
	if *prefixPassed {
		activePrefix = "[gsub] "

		args := os.Args[1:]
		for i, arg := range args {
			if arg == "-p" || arg == "--prefix" || strings.HasPrefix(arg, "-p=") || strings.HasPrefix(arg, "--prefix=") {
				var potentialValue string

				if strings.Contains(arg, "=") {
					parts := strings.SplitN(arg, "=", 2)
					potentialValue = parts[1]
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

	// 4. Load Environment Variables / Stream
	fileVars := make(map[string]string)
	if *envFilePath != "" {
		var reader io.Reader
		if *envFilePath == "-" {
			if *templatePath == "" {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=deadlock message=\"cannot read env and template from STDIN simultaneously\"\n", activePrefix)
				os.Exit(1)
			}
			reader = os.Stdin
		} else {
			file, err := os.Open(*envFilePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%sstate=failed reason=file_missing target=\"%s\"\n", activePrefix, *envFilePath)
				os.Exit(1)
			}
			defer file.Close()
			reader = file
		}

		vars, err := parseEnvReader(reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sstate=failed reason=parse_error message=\"%v\"\n", activePrefix, err)
			os.Exit(1)
		}
		fileVars = vars
	}

	// 5. Select Template Source
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
		templateReader = os.Stdin
	}

	// 6. Processing Logic (Atomic In-Memory Output Assembly)
	var outBuffer bytes.Buffer
	missingVariables := make(map[string]bool)

	placeholderRegex := regexp.MustCompile(`(\\)?\{\{\s*((?:\$[a-zA-Z0-9]+\.)?[a-zA-Z0-9._-]+)\s*(?:\|\|\s*'([^']*)')?\s*\}\}`)

	scanner := bufio.NewScanner(templateReader)

	for scanner.Scan() {
		line := scanner.Text()
		processedLine := placeholderRegex.ReplaceAllStringFunc(line, func(match string) string {
			submatches := placeholderRegex.FindStringSubmatch(match)
			isEscaped := submatches[1] != ""
			fullKey := submatches[2]
			hasDefault := len(submatches) > 3 && submatches[3] != "" || strings.Contains(match, "||")
			defaultValue := ""
			if len(submatches) > 3 {
				defaultValue = submatches[3]
			}

			if isEscaped {
				if hasDefault {
					return "{{" + fullKey + " || '" + defaultValue + "'}}"
				}
				return "{{" + fullKey + "}}"
			}

			var val string
			switch {
			case strings.HasPrefix(fullKey, "$env."):
				realKey := strings.TrimPrefix(fullKey, "$env.")
				val = os.Getenv(realKey)

			case strings.HasPrefix(fullKey, "$file.") || strings.HasPrefix(fullKey, "$input."):
				realKey := fullKey[strings.Index(fullKey, ".")+1:]
				val = fileVars[realKey]

			default:
				if *useEnv {
					val = os.Getenv(fullKey)
				}
				if val == "" {
					val = fileVars[fullKey]
				}
			}

			if val == "" {
				if hasDefault {
					return defaultValue
				}
				missingVariables[fullKey] = true
				return match
			}
			return val
		})

		// Append the processed line into our isolated internal buffer channel
		outBuffer.WriteString(processedLine + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%sstate=failed reason=stream_error message=\"%v\"\n", activePrefix, err)
		os.Exit(1)
	}

	// 7. Logfmt Safety Validation & Firewalled Output Commitment
	if len(missingVariables) > 0 && !*allowMissing {
		keys := make([]string, 0, len(missingVariables))
		for k := range missingVariables {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		targets := strings.Join(keys, ",")

		// Emit structural diagnostics safely to STDERR
		fmt.Fprintf(os.Stderr, "%sstate=failed reason=missing_vars targets=%s fix=--allow-missing\n", activePrefix, targets)
		os.Exit(1) // Terminates execution immediately; the outBuffer is discarded safely
	}

	// Everything passed validation cleanly. Write the entire buffer contents out to STDOUT.
	_, _ = outBuffer.WriteTo(os.Stdout)
}

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
