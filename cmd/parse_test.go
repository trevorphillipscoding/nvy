package cmd

import (
	"testing"
)

func TestParseToolVersion_TwoArgs(t *testing.T) {
	cases := []struct {
		args     []string
		wantTool string
		wantVer  string
	}{
		{[]string{"go", "1.22.1"}, "go", "1.22.1"},
		{[]string{"node", "20.11.1"}, "node", "20.11.1"},
		{[]string{"python", "3.12.5"}, "python", "3.12.5"},
		{[]string{" go ", " 1.22.1 "}, "go", "1.22.1"}, // whitespace trimmed
		{[]string{"go", "1.26."}, "go", "1.26"},        // trailing dot stripped
		{[]string{"go", "1.26..."}, "go", "1.26"},      // multiple trailing dots
	}
	for _, c := range cases {
		tool, ver, err := parseToolVersion(c.args)
		if err != nil {
			t.Errorf("parseToolVersion(%v): unexpected error: %v", c.args, err)
			continue
		}
		if tool != c.wantTool {
			t.Errorf("parseToolVersion(%v) tool = %q; want %q", c.args, tool, c.wantTool)
		}
		if ver != c.wantVer {
			t.Errorf("parseToolVersion(%v) ver = %q; want %q", c.args, ver, c.wantVer)
		}
	}
}

func TestParseToolVersion_AtSyntax(t *testing.T) {
	cases := []struct {
		args     []string
		wantTool string
		wantVer  string
	}{
		{[]string{"go@1.22.1"}, "go", "1.22.1"},
		{[]string{"node@20.11.1"}, "node", "20.11.1"},
		{[]string{"python@3.12.5+20240814"}, "python", "3.12.5+20240814"},
	}
	for _, c := range cases {
		tool, ver, err := parseToolVersion(c.args)
		if err != nil {
			t.Errorf("parseToolVersion(%v): unexpected error: %v", c.args, err)
			continue
		}
		if tool != c.wantTool {
			t.Errorf("parseToolVersion(%v) tool = %q; want %q", c.args, tool, c.wantTool)
		}
		if ver != c.wantVer {
			t.Errorf("parseToolVersion(%v) ver = %q; want %q", c.args, ver, c.wantVer)
		}
	}
}

func TestParseToolVersion_Errors(t *testing.T) {
	cases := []struct {
		args []string
		desc string
	}{
		{[]string{"go"}, "missing version"},
		{[]string{"@1.22.1"}, "missing tool"},
		{[]string{"go@"}, "missing version after @"},
		{[]string{""}, "empty string"},
	}
	for _, c := range cases {
		_, _, err := parseToolVersion(c.args)
		if err == nil {
			t.Errorf("parseToolVersion(%v) [%s]: expected error, got nil", c.args, c.desc)
		}
	}
}
