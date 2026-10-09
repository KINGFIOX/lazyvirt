// lazyvirt: a lazygit-style TUI for managing libvirt domains, backed by the
// virsh CLI.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"lazyvirt/internal/tui"
	"lazyvirt/internal/virsh"
)

// version is injected at build time: -ldflags "-X main.version=<git describe>".
var version = "dev"

func main() {
	listMode := flag.Bool("list", false, "non-interactive mode: print the parsed domain list and exit (smoke test)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("lazyvirt", version)
		return
	}

	if _, err := exec.LookPath("virsh"); err != nil {
		fatal("virsh not found in PATH, please install libvirt first")
	}

	if *listMode {
		listOnly()
		return
	}

	p := tea.NewProgram(tui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatal("TUI failed: " + err.Error())
	}
}

func listOnly() {
	domains, err := virsh.ListAll()
	if err != nil {
		fatal(err.Error())
	}
	if len(domains) == 0 {
		fmt.Println("(no domains)")
		return
	}
	fmt.Printf("%-6s%-20s%-16s%s\n", "ID", "NAME", "STATE", "AUTOSTART")
	for _, d := range domains {
		fmt.Printf("%-6s%-20s%-16s%s\n", d.ID, d.Name, d.State, d.Autostart)
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error: "+msg)
	os.Exit(1)
}
