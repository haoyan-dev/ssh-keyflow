package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"ssh-keyflow/internal/state"
)

type Command struct {
	Name        string
	Args        []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool
}

type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, Command) error
}

type SystemRunner struct{}

func (SystemRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (SystemRunner) Run(ctx context.Context, command Command) error {
	if command.Interactive {
		terminal, valid := command.Stdin.(*os.File)
		if !valid {
			return fmt.Errorf("a real terminal is required for credential input")
		}
		info, err := terminal.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return fmt.Errorf("a real terminal is required for credential input")
		}
	}
	process := exec.CommandContext(ctx, command.Name, command.Args...)
	process.Stdin, process.Stdout, process.Stderr = command.Stdin, command.Stdout, command.Stderr
	return process.Run()
}

type Options struct {
	Home   string
	Input  io.Reader
	Output io.Writer
	Runner Runner
}

func Run(ctx context.Context, options Options) error {
	if options.Home == "" {
		return fmt.Errorf("Home is required")
	}
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Runner == nil {
		options.Runner = SystemRunner{}
	}
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	if err := integrity(document); err != nil {
		fmt.Fprintln(options.Output, err, "(updates stopped)")
	}
	if err := state.CheckWritable(options.Home); err != nil {
		return err
	}
	for {
		missing := false
		for _, name := range []string{"ssh", "ssh-keygen"} {
			if _, err := options.Runner.LookPath(name); err != nil {
				missing = true
				fmt.Fprintf(options.Output, "Missing %s: %v\n", name, err)
			}
		}
		if !missing {
			break
		}
		fmt.Fprintln(options.Output, "Windows (elevated PowerShell): Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0\nLinux (Debian/Ubuntu): sudo apt install openssh-client; Fedora: sudo dnf install openssh-clients\nmacOS: xcode-select --install; use the system OpenSSH client\nr: Recheck; 0: Exit")
		answer, err := readLine(options.Input)
		if err != nil || answer != "r" {
			return fmt.Errorf("required OpenSSH commands are unavailable")
		}
	}
	for {
		fmt.Fprintln(options.Output, "SSH KeyFlow\n1: Add destination and register\n2: List destinations and keys\n3: Register another public key\n4: Unregister public key\n5: Create or import key\n6: Delete local key\n0: Exit")
		choice, err := readLine(options.Input)
		if err == io.EOF || choice == "0" {
			return nil
		}
		if err != nil {
			return err
		}
		switch choice {
		case "1":
			if err := onboard(ctx, options); err != nil {
				fmt.Fprintln(options.Output, "Onboarding stopped:", err)
			}
		case "2":
			document, err := state.Load(options.Home)
			if err != nil {
				return err
			}
			if err := integrity(document); err != nil {
				fmt.Fprintln(options.Output, err, "(updates stopped)")
			}
			list(options, document)
		case "3":
			if err := registerAnother(ctx, options); err != nil {
				fmt.Fprintln(options.Output, "Registration stopped:", err)
			}
		case "5":
			if err := importKey(ctx, options); err != nil {
				fmt.Fprintln(options.Output, "Import stopped:", err)
			}
		default:
			fmt.Fprintln(options.Output, "Not implemented in this build; no changes made")
		}
	}
}

func readLine(input io.Reader) (string, error) {
	var line strings.Builder
	buffer := make([]byte, 1)
	for {
		count, err := input.Read(buffer)
		if count > 0 {
			if buffer[0] == '\n' {
				return strings.TrimSpace(line.String()), nil
			}
			line.WriteByte(buffer[0])
		}
		if err != nil {
			return strings.TrimSpace(line.String()), err
		}
	}
}
