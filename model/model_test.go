// /home/krylon/go/src/github.com/blicero/guangng/model/model_test.go
// -*- mode: go; coding: utf-8; -*-
// Created on 20. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-20 13:32:26 krylon>

package model

import "testing"

func TestHostZone(t *testing.T) {
	type testCase struct {
		name     string
		expected string
	}

	var testcases = []testCase{
		{"www.example.com.", "example.com"},
		{"www.", ""},
		{"mail.wtf", "wtf"},
	}

	for _, c := range testcases {
		var h = Host{Name: c.name}
		var zone = h.Zone()

		if zone != c.expected {
			t.Errorf("Unexpected result from Host.Zone: %s (expected %s)",
				zone,
				c.expected)
		}
	}
} // func TestHostZone(t *testing.T)
