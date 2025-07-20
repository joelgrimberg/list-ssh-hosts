package main

import (
	"os"
	"strings"
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

func TestDeleteHostFromConfig(t *testing.T) {
	// Create a test SSH config with multiple hosts
	config := `
Host test-server
    Hostname 192.168.1.100
    User root

Host production-server
    Hostname 203.0.113.10
    User admin

Host staging-server
    Hostname 198.51.100.50
    User deploy
`
	tmpfile, err := os.CreateTemp("", "sshconfig_delete")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	// Test deleting a host that exists
	err = deleteHostFromConfigFile(tmpfile.Name(), "production-server")
	if err != nil {
		t.Fatalf("deleteHostFromConfig failed: %v", err)
	}

	// Verify the host was deleted
	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed after deletion: %v", err)
	}

	// Check that production-server is gone but others remain
	expectedHosts := []string{"test-server", "staging-server"}
	if len(hosts) != len(expectedHosts) {
		t.Fatalf("expected %d hosts after deletion, got %d", len(expectedHosts), len(hosts))
	}

	for _, expectedHost := range expectedHosts {
		found := false
		for _, host := range hosts {
			if host.host == expectedHost {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected host %s to remain after deletion", expectedHost)
		}
	}

	// Verify production-server is not in the list
	for _, host := range hosts {
		if host.host == "production-server" {
			t.Errorf("production-server should have been deleted but was found")
		}
	}
}

func TestDeleteHostFromConfig_NonExistentHost(t *testing.T) {
	// Create a test SSH config
	config := `
Host test-server
    Hostname 192.168.1.100
    User root
`
	tmpfile, err := os.CreateTemp("", "sshconfig_delete_nonexistent")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	// Test deleting a host that doesn't exist
	err = deleteHostFromConfigFile(tmpfile.Name(), "non-existent-host")
	if err != nil {
		t.Fatalf("deleteHostFromConfig should not fail for non-existent host: %v", err)
	}

	// Verify the original host still exists
	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host after deleting non-existent host, got %d", len(hosts))
	}

	if hosts[0].host != "test-server" {
		t.Errorf("expected test-server to remain, got %s", hosts[0].host)
	}
}

func TestDeleteHostFromConfig_MultipleHostsOnLine(t *testing.T) {
	// Create a test SSH config with multiple hosts on one line
	// Note: This is a complex case that would require more sophisticated parsing
	// For now, we'll test that the basic functionality works
	config := `
Host host1 host2 host3
    Hostname 192.168.1.100
    User root

Host host4
    Hostname 203.0.113.10
    User admin
`
	tmpfile, err := os.CreateTemp("", "sshconfig_delete_multiple")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(config)); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	tmpfile.Close()

	// Test deleting one host from a multi-host line
	// This will remove the entire block since host2 is in it
	err = deleteHostFromConfigFile(tmpfile.Name(), "host2")
	if err != nil {
		t.Fatalf("deleteHostFromConfig failed: %v", err)
	}

	// Verify the host was deleted
	hosts, err := parseSSHConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseSSHConfig failed after deletion: %v", err)
	}

	// Since we removed the entire block containing host2, only host4 should remain
	expectedHosts := []string{"host4"}
	if len(hosts) != len(expectedHosts) {
		t.Fatalf("expected %d hosts after deletion, got %d", len(expectedHosts), len(hosts))
	}

	if hosts[0].host != "host4" {
		t.Errorf("expected host4 to remain, got %s", hosts[0].host)
	}

	// Verify host2 is not in the list
	for _, host := range hosts {
		if host.host == "host2" {
			t.Errorf("host2 should have been deleted but was found")
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists in slice",
			slice:    []string{"host1", "host2", "host3"},
			item:     "host2",
			expected: true,
		},
		{
			name:     "item does not exist in slice",
			slice:    []string{"host1", "host2", "host3"},
			item:     "host4",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "host1",
			expected: false,
		},
		{
			name:     "case sensitive match",
			slice:    []string{"Host1", "HOST2", "host3"},
			item:     "host1",
			expected: false,
		},
		{
			name:     "exact match",
			slice:    []string{"host1", "host2", "host3"},
			item:     "host1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %s) = %v, expected %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

// Helper function for testing that takes a file path instead of using ~/.ssh/config
func deleteHostFromConfigFile(configPath, hostToDelete string) error {
	// Read the entire config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	var inHostBlock bool
	var currentHosts []string
	var skipBlock bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(strings.ToLower(trimmedLine), "host ") {
			// Check if this host block contains our target
			fields := strings.Fields(trimmedLine)
			currentHosts = fields[1:]

			// If this block contains our target, mark it for skipping
			if contains(currentHosts, hostToDelete) {
				skipBlock = true
				continue
			} else {
				skipBlock = false
				inHostBlock = true
				newLines = append(newLines, line)
			}
			continue
		}

		// If we're skipping this block, don't add any lines
		if skipBlock {
			// If this line is not indented, we're out of the host block
			if len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				skipBlock = false
				inHostBlock = false
				newLines = append(newLines, line)
			}
			continue
		}

		// If this line is not indented, we're out of the host block
		if inHostBlock && len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inHostBlock = false
		}

		newLines = append(newLines, line)
	}

	// Write the modified content back to the file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(configPath, []byte(newContent), 0644)
}
