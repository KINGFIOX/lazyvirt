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

// ExecLog reports one virsh invocation for display in the Command Log.
type ExecLog struct {
	Cmdline string // full echo, e.g. "virsh start demo"
	Output  string // raw stdout
	Err     error  // nil on success; message is the raw stderr or timeout
}

// OnExec, when set, is invoked after every non-quiet Run call. Set by the UI
// to funnel invocations into the Command Log panel.
var OnExec func(ExecLog)

type quietKey struct{}

// Quiet marks a context so that Run calls made under it are not reported to
// OnExec (used for the periodic background refresh).
func Quiet(ctx context.Context) context.Context {
	return context.WithValue(ctx, quietKey{}, true)
}

const cmdTimeout = 30 * time.Second

// Run executes one virsh subcommand and returns stdout; on failure the error
// message is the raw stderr. Non-quiet invocations are reported via OnExec.
func Run(ctx context.Context, args ...string) (string, error) {
	cmdline := "virsh " + strings.Join(args, " ")
	tctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(tctx, "virsh", args...)
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()

	var repErr error
	if err != nil {
		switch {
		case tctx.Err() == context.DeadlineExceeded:
			repErr = fmt.Errorf("%s: timed out after %s", cmdline, cmdTimeout)
		case strings.TrimSpace(errb.String()) != "":
			repErr = fmt.Errorf("%s", strings.TrimSpace(errb.String()))
		default:
			repErr = err
		}
	}

	if OnExec != nil && ctx.Value(quietKey{}) == nil {
		OnExec(ExecLog{Cmdline: cmdline, Output: out.String(), Err: repErr})
	}
	return out.String(), repErr
}

// ListAll corresponds to `virsh list --all`, additionally fetching each
// domain's autostart state concurrently (the per-domain dominfo lookups run
// quiet so they do not flood the Command Log).
func ListAll(ctx context.Context) ([]Domain, error) {
	out, err := Run(ctx, "list", "--all")
	if err != nil {
		return nil, err
	}
	domains := parseList(out)
	qctx := Quiet(ctx)
	var wg sync.WaitGroup
	for i := range domains {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			info, err := Run(qctx, "dominfo", domains[i].Name)
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
func DomInfo(ctx context.Context, name string) (Detail, error) {
	out, err := Run(ctx, "dominfo", name)
	if err != nil {
		return Detail{}, err
	}
	return parseDomInfoOrdered(out), nil
}

func Start(ctx context.Context, name string) (string, error) {
	return Run(ctx, "start", name)
}

func Shutdown(ctx context.Context, name string) (string, error) {
	return Run(ctx, "shutdown", name)
}

func Reboot(ctx context.Context, name string) (string, error) {
	return Run(ctx, "reboot", name)
}

func Destroy(ctx context.Context, name string) (string, error) {
	return Run(ctx, "destroy", name)
}

// AutostartSet corresponds to `virsh autostart <name>` (enable=true)
// or `virsh autostart <name> --disable` (enable=false).
func AutostartSet(ctx context.Context, name string, enable bool) (string, error) {
	if enable {
		return Run(ctx, "autostart", name)
	}
	return Run(ctx, "autostart", name, "--disable")
}

// Undefine corresponds to `virsh undefine <name>`;
// nvram also removes the UEFI var store, removeAllStorage removes all
// associated disk files.
func Undefine(ctx context.Context, name string, nvram, removeAllStorage bool) (string, error) {
	args := []string{"undefine", name}
	if nvram {
		args = append(args, "--nvram")
	}
	if removeAllStorage {
		args = append(args, "--remove-all-storage")
	}
	return Run(ctx, args...)
}
