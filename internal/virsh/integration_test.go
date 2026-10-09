//go:build integration

// Integration tests against the real libvirtd:
// `go test -tags integration ./internal/virsh/`.
// Creates a throwaway domain lazyvirt-selftest; never touches existing domains.
package virsh

import (
	"os"
	"path/filepath"
	"testing"
)

const testDomain = "lazyvirt-selftest"

const testXML = `<domain type='kvm'>
  <name>lazyvirt-selftest</name>
  <memory unit='MiB'>64</memory>
  <vcpu>1</vcpu>
  <os><type arch='x86_64'>hvm</type></os>
  <devices><console type='pty'/></devices>
</domain>`

func findDomain(t *testing.T, name string) Domain {
	t.Helper()
	domains, err := ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	for _, d := range domains {
		if d.Name == name {
			return d
		}
	}
	return Domain{}
}

func TestIntegrationFlow(t *testing.T) {
	Undefine(testDomain, true, false) // clean up leftovers from a previous run, ignore errors

	t.Cleanup(func() {
		Undefine(testDomain, true, false)
		Destroy(testDomain) // if start succeeded, make sure it is off
	})

	xmlPath := filepath.Join(t.TempDir(), "test.xml")
	if err := os.WriteFile(xmlPath, []byte(testXML), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run("define", xmlPath); err != nil {
		t.Fatalf("define: %v", err)
	}

	// List: present, shut off, autostart=disable
	if d := findDomain(t, testDomain); d.Name == "" {
		t.Fatal("defined domain not found in ListAll")
	} else {
		if d.State != "shut off" {
			t.Errorf("state = %q, want shut off", d.State)
		}
		if d.Autostart != "disable" {
			t.Errorf("autostart = %q, want disable", d.Autostart)
		}
	}

	// autostart toggle both ways
	if _, err := AutostartSet(testDomain, true); err != nil {
		t.Fatalf("autostart enable: %v", err)
	}
	if info, _ := DomInfo(testDomain); info.Get("Autostart") != "enable" {
		t.Errorf("dominfo autostart = %q, want enable", info.Get("Autostart"))
	}
	if _, err := AutostartSet(testDomain, false); err != nil {
		t.Fatalf("autostart disable: %v", err)
	}
	if info, _ := DomInfo(testDomain); info.Get("Autostart") != "disable" {
		t.Errorf("dominfo autostart = %q, want disable", info.Get("Autostart"))
	}

	// DomInfo: key fields present
	if info, err := DomInfo(testDomain); err != nil {
		t.Fatalf("DomInfo: %v", err)
	} else if info.Get("Name") != testDomain || info.Get("State") != "shut off" {
		t.Errorf("DomInfo Name/State = %q/%q", info.Get("Name"), info.Get("State"))
	}

	// start + destroy (a diskless domain reaching firmware counts as started;
	// both outcomes are accepted)
	if _, err := Start(testDomain); err == nil {
		if _, err := Destroy(testDomain); err != nil {
			t.Fatalf("destroy: %v", err)
		}
	}

	// gone from the list after undefine
	if _, err := Undefine(testDomain, false, false); err != nil {
		t.Fatalf("undefine: %v", err)
	}
	if d := findDomain(t, testDomain); d.Name != "" {
		t.Error("domain still present after undefine")
	}
}
