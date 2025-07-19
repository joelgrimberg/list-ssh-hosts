package main

import (
	"os"
	"testing"
)

func TestParseSSHConfig(t *testing.T) {
	config := `
Host test-server
    Hostname 192.168.1.100
    User root

HOST production-server
    Hostname 203.0.113.10
    User admin

Host staging-server
    Hostname 198.51.100.50
    User deploy

Host onlyip
    Hostname 2.2.2.2

Host onlyuser
    User admin

Host *
    ForwardAgent yes

Host wildcard-?
    Hostname 3.3.3.3
    User admin
`
	tmpfile, err := os.CreateTemp("", "sshconfig")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}

	expected := []struct {
		host string
		desc string
	}{
		{"test-server", "root@192.168.1.100"},
		{"production-server", "admin@203.0.113.10"},
		{"staging-server", "deploy@198.51.100.50"},
		{"onlyip", "2.2.2.2"},
		{"onlyuser", ""},
	}
	if len(hosts) != len(expected) {
		t.Fatalf("expected %d hosts, got %d", len(expected), len(hosts))
	}
	for i, exp := range expected {
		if hosts[i].host != exp.host {
			t.Errorf("expected host %q, got %q", exp.host, hosts[i].host)
		}
		if hosts[i].desc != exp.desc {
			t.Errorf("expected desc %q, got %q", exp.desc, hosts[i].desc)
		}
	}
}

func TestParseSSHConfig_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "sshconfig_empty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseSSHConfig_OnlyWildcards(t *testing.T) {
	config := `
Host *
    Hostname 1.2.3.4
Host ?
    Hostname 2.3.4.5
Host [abc]
    Hostname 3.4.5.6
`
	tmpfile, err := os.CreateTemp("", "sshconfig_wildcards")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseSSHConfig_MultipleHostsOnLine(t *testing.T) {
	config := `
Host host1 host2 host3
    Hostname 1.2.3.4
`
	tmpfile, err := os.CreateTemp("", "sshconfig_multi")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	expected := []string{"host1", "host2", "host3"}
	if len(hosts) != len(expected) {
		t.Fatalf("expected %d hosts, got %d", len(expected), len(hosts))
	}
	for i, h := range expected {
		if hosts[i].host != h {
			t.Errorf("expected host %q, got %q", h, hosts[i].host)
		}
	}
}

func TestParseSSHConfig_NoHostname(t *testing.T) {
	config := `
Host noiphost
    User root
`
	tmpfile, err := os.CreateTemp("", "sshconfig_noip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].host != "noiphost" {
		t.Errorf("expected host 'noiphost', got %q", hosts[0].host)
	}
	if hosts[0].desc != "" {
		t.Errorf("expected empty desc, got %q", hosts[0].desc)
	}
}

func TestParseSSHConfig_WithHostnameAndUser(t *testing.T) {
	config := `
Host iphost
    Hostname 10.0.0.1
    User admin
`
	tmpfile, err := os.CreateTemp("", "sshconfig_withipuser")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].host != "iphost" {
		t.Errorf("expected host 'iphost', got %q", hosts[0].host)
	}
	if hosts[0].desc != "admin@10.0.0.1" {
		t.Errorf("expected desc 'admin@10.0.0.1', got %q", hosts[0].desc)
	}
}

func TestParseSSHConfig_WithHostnameOnly(t *testing.T) {
	config := `
Host iponly
    Hostname 10.0.0.2
`
	tmpfile, err := os.CreateTemp("", "sshconfig_withiponly")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].host != "iponly" {
		t.Errorf("expected host 'iponly', got %q", hosts[0].host)
	}
	if hosts[0].desc != "10.0.0.2" {
		t.Errorf("expected desc '10.0.0.2', got %q", hosts[0].desc)
	}
}

func TestParseSSHConfig_FileNotExist(t *testing.T) {
	_, err := parseSSHConfig("/tmp/this_file_should_not_exist_1234567890")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}
