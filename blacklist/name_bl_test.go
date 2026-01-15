// /home/krylon/go/src/github.com/blicero/guangng/blacklist/name_bl_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 12. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-15 18:53:47 krylon>

package blacklist

import "testing"

var nbl *BlacklistName

func TestCreateNameBlacklist(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			nbl = nil
			t.Fatalf("Failed to create blacklist: %s", x)
		}
	}()

	nbl = NewBlacklistName()

	if nbl == nil {
		t.Fatalf("NewNameBlacklist returned nil!")
	} else if len(nbl.items) != len(defaultNamePatterns) {
		t.Fatalf("BlacklistName has unexpected length: %d (expected %d)",
			len(nbl.items),
			len(defaultNamePatterns))
	}
} // func TestCreateNameBlacklist(t *testing.T)

func TestBlacklistNameMatch(t *testing.T) {
	type nameTestCase struct {
		name   string
		expRes bool
	}

	var testCases = []nameTestCase{
		{name: "www.my-cool-domain.com"},
		{"customer-wiki.my-cool-domain.com", true},
	}

	if nbl == nil {
		t.SkipNow()
	}

	for _, c := range testCases {
		if m := nbl.Match(c.name); m != c.expRes {
			t.Errorf("Unexpected result for item %s: %t (expected %t)",
				c.name,
				m,
				c.expRes)
		}
	}
} // func TestBlacklistNameMatch(t *testing.T)
