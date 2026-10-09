// Package virsh wraps the virsh CLI: each feature maps to one virsh
// subcommand, output is returned as structs, and errors carry the raw stderr.
package virsh

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Domain is one row of `virsh list --all`.
type Domain struct {
	ID        string // the Id column, "-" for inactive domains
	Name      string
	State     string // running / paused / shut off / in shutdown / crashed / pmsuspended ...
	Autostart string // from dominfo: enable / disable / unknown
}

// DetailLine is one "Key: value" line of dominfo output.
type DetailLine struct {
	Key string
	Val string
}

// Detail is the parsed result of `virsh dominfo`, preserving original order.
type Detail struct {
	Lines []DetailLine
}

// Get returns the value for key, or "" if absent.
func (d Detail) Get(key string) string {
	for _, l := range d.Lines {
		if l.Key == key {
			return l.Val
		}
	}
	return ""
}

const cmdTimeout = 30 * time.Second

// Run executes one virsh subcommand and returns stdout; on failure the error
// message is the raw stderr. Exported so callers can compose commands
// (TUI async commands, the --list debug mode).
func Run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "virsh", args...)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("virsh %s: timed out after %s", strings.Join(args, " "), cmdTimeout)
		}
		if stderr := strings.TrimSpace(errb.String()); stderr != "" {
			return "", fmt.Errorf("%s", stderr)
		}
		return "", err
	}
	return out.String(), nil
}

// ListAll corresponds to `virsh list --all`, additionally fetching each
// domain's autostart state concurrently.
func ListAll() ([]Domain, error) {
	out, err := Run("list", "--all")
	if err != nil {
		return nil, err
	}
	domains := parseList(out)
	var wg sync.WaitGroup
	for i := range domains {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			info, err := Run("dominfo", domains[i].Name)
			if err != nil {
				domains[i].Autostart = "unknown"
				return
			}
			domains[i].Autostart = parseDomInfoMap(info)["Autostart"]
		}(i)
	}
	wg.Wait()
	return domains, nil
}

// DomInfo corresponds to `virsh dominfo <name>`.
func DomInfo(name string) (Detail, error) {
	out, err := Run("dominfo", name)
	if err != nil {
		return Detail{}, err
	}
	return parseDomInfoOrdered(out), nil
}

func Start(name string) (string, error)    { return Run("start", name) }
func Shutdown(name string) (string, error) { return Run("shutdown", name) }
func Reboot(name string) (string, error)   { return Run("reboot", name) }
func Destroy(name string) (string, error)  { return Run("destroy", name) }

// AutostartSet corresponds to `virsh autostart <name>` (enable=true)
// or `virsh autostart <name> --disable` (enable=false).
func AutostartSet(name string, enable bool) (string, error) {
	if enable {
		return Run("autostart", name)
	}
	return Run("autostart", name, "--disable")
}

// Undefine corresponds to `virsh undefine <name>`;
// nvram also removes the UEFI var store, removeAllStorage removes all
// associated disk files.
func Undefine(name string, nvram, removeAllStorage bool) (string, error) {
	args := []string{"undefine", name}
	if nvram {
		args = append(args, "--nvram")
	}
	if removeAllStorage {
		args = append(args, "--remove-all-storage")
	}
	return Run(args...)
}
