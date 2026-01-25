// /home/krylon/go/src/github.com/blicero/guangng/scanner/probe.go
// -*- mode: go; coding: utf-8; -*-
// Created on 24. 01. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-01-25 18:57:19 krylon>

package scanner

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/alouca/gosnmp"
	"github.com/blicero/guangng/common"
	"github.com/blicero/guangng/model"
	dns "github.com/tonnerre/golang-dns"
)

func (scn *Scanner) probePort(host *model.Host, port uint16) (*scanResult, error) {
	var (
		err error
		res *scanResult
	)

	switch port {
	case 21, 22, 25, 110, 143, 2525:
		// simple plaintext scan
		return scn.scanPlain(host, port)
	case 23, 3270, 9023:
		return scn.scanTelnet(host, port)
	case 53, 5353:
		return scn.scanDNS(host, port)
	case 79:
		return scn.scanFinger(host, port)
	case 80, 443, 8000, 8080:
		return scn.scanHTTP(host, port)
	case 161:
		return scn.scanSNMP(host, port)
	}

	return res, err
} // func (scn *Scanner) probePort(host *model.Host, port uint16) (*scanResult, error)

func (scn *Scanner) scanPlain(host *model.Host, port uint16) (*scanResult, error) {
	scn.log.Printf("[TRACE] Scanning %s:%d using plain scanner.\n", host.AStr(), port)
	var (
		err error
		res = &scanResult{
			host: host,
			svc: &model.Service{
				HostID:    host.ID,
				Port:      port,
				Timestamp: time.Now(),
			},
		}
		srv    = fmt.Sprintf("[%s]:%d", host.AStr(), port)
		conn   net.Conn
		reader *bufio.Reader
		line   string
	)

	if conn, err = net.Dial("tcp", srv); err != nil {
		err = fmt.Errorf("error connecting to %s: %w", srv, err)
		goto END
	}

	defer conn.Close() // nolint: errcheck

	reader = bufio.NewReader(conn)
	if line, err = reader.ReadString('\n'); err != nil {
		err = fmt.Errorf("error receiving data from %s: %w", srv, err)
		goto END
	}

	line = newline.ReplaceAllString(line, "")
	scn.log.Printf("[TRACE] Got reply from %s:%d : %s\n",
		host.AStr(),
		port,
		line)

	res.svc.Response = line
	res.svc.Success = true

END:
	return res, err
} // func (scn *Scanner) scanPlain(host *model.Host, port uint16) (*scanResult, error)

func (scn *Scanner) scanFinger(host *model.Host, port uint16) (*scanResult, error) {
	var (
		err        error
		recvbuffer = make([]byte, 4096)
		n          int
	)
	const TIMEOUT = 5 * time.Second

	scn.log.Printf("[TRACE] Fingering root@%s (port %d)...\n",
		host.Name, port)

	srv := fmt.Sprintf("[%s]:%d", host.AStr(), port)
	conn, err := net.Dial("tcp", srv)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to %s: %s", srv, err.Error())
		return nil, errors.New(msg)
	}

	defer conn.Close() // nolint: errcheck

	conn.Write([]byte("root\r\n")) // nolint: errcheck

	conn.SetDeadline(time.Now().Add(TIMEOUT)) // nolint: errcheck

	if n, err = conn.Read(recvbuffer); err != nil {
		msg := fmt.Sprintf("Error receiving from [%s]:%d - %s",
			host.AStr(), port, err.Error())
		return nil, errors.New(msg)
	}

	result := &scanResult{
		host: host,
		svc: &model.Service{
			Port:      port,
			Response:  string(recvbuffer[:n]),
			Timestamp: time.Now(),
			Success:   true,
		},
	}
	return result, nil
} // func (scn *Scanner) scanFinger(host *model.Host, port uint16) (*scanResult, error)

var dnsReplyPat *regexp.Regexp = regexp.MustCompile("\"([^\"]+)\"") // nolint: unused

// Samstag, 05. 07. 2014, 20:26
// Den Code habe ich mehr oder weniger aus dem Beispiel im golang-dns Repository
// geklaut, Copyright 2011 Miek Gieben
//
// Samstag, 26. 07. 2014, 13:22
// Kann es sein, dass das nicht ganz so funktioniert, wie ich mir das vorstelle?
// Ich bekomme irgendwie nicht einen einzigen Port 53 erfolgreich gescannt...
//
// Ich habe den Quellcode kritisch angestarrt und keinen offensichtlichen Fehler
// entdeckt. Ich sollte mal testen, ob das Ding überhaupt funktioniert.
//
// Freitag, 01. 08. 2014, 17:59
// Mmmh, es gibt da ein kleines Problem: Die Replies, die in der Datenbank landen, sehen ungefähr so aus:
// version.bind.   1476526080      IN      TXT     "Microsoft DNS 6.1.7601 (1DB14556)"

func (scn *Scanner) scanDNS(host *model.Host, port uint16) (*scanResult, error) {
	scn.log.Printf("Scanning %s:%d using DNS scanner.\n", host.AStr(), port)

	m := new(dns.Msg)
	m.Question = make([]dns.Question, 1)
	c := new(dns.Client)
	m.Question[0] = dns.Question{Name: "version.bind.", Qtype: dns.TypeTXT, Qclass: dns.ClassCHAOS}
	addr := fmt.Sprintf("[%s]:%d", host.AStr(), port)
	in, _, err := c.Exchange(m, addr)
	if err != nil {
		msg := fmt.Sprintf("Error asking %s for version.bind: %s", host.Name, err.Error())
		return nil, errors.New(msg)
	} else if in != nil && len(in.Answer) > 0 {
		reply := in.Answer[0]
		switch t := reply.(type) {
		case *dns.TXT:
			versionStr := new(string)
			*versionStr = t.String()
			match := dnsReplyPat.FindStringSubmatch(*versionStr)
			if nil != match {
				*versionStr = match[1]
			}

			var result = &scanResult{
				host: host,
				svc: &model.Service{
					HostID:    host.ID,
					Port:      port,
					Response:  *versionStr,
					Success:   true,
					Timestamp: time.Now(),
				},
			}

			scn.log.Printf("[DEBUG] Got reply: %s:%d is %s\n",
				host.AStr(),
				port,
				*versionStr)
			return result, nil
		default:
			// CANTHAPPEN
			println("Potzblitz! Damit konnte ja wirklich NIEMAND rechnen!")
			return nil, errors.New("no reply was received")
		}
	}

	return nil, errors.New("no valid reply was received, but error status was nil")
} // func (scn *Scanner) scanDNS(host *model.Host, port uint16) (*scanResult, error)

func (scn *Scanner) scanHTTP(host *model.Host, port uint16) (*scanResult, error) {
	if host == nil {
		return nil, errors.New("host is nil")
	} else if common.Debug {
		scn.log.Printf("[DEBUG] Scanning %s:%d using HTTP scanner.\n", host.AStr(), port)
	}

	transport := &http.Transport{
		Proxy: nil,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   common.ActiveTimeout * 2,
	}

	var schema = "http"
	if port == 443 {
		schema += "s"
	}
	url := fmt.Sprintf("%s://%s:%d/", schema, host.AStr(), port)
	response, err := client.Head(url)
	if err != nil {
		msg := fmt.Sprintf("Error fetching headers for URL %s: %s", url, err.Error())
		return nil, errors.New(msg)
	}

	var result = &scanResult{
		host: host,
		svc: &model.Service{
			HostID:    host.ID,
			Port:      port,
			Response:  newline.ReplaceAllString(response.Header.Get("Server"), ""),
			Timestamp: time.Now(),
		},
	}

	scn.log.Printf("[TRACE] http://%s:%d/ -> %s\n",
		host.AStr(),
		port,
		result.svc.Response)
	return result, nil
} // func (scn *Scanner) scanHTTP(host *model.Host, port uint16) (*scanResult, error)

func (scn *Scanner) scanSNMP(host *model.Host, port uint16) (*scanResult, error) {
	scn.log.Printf("[TRACE] Scanning %s:%d using SNMP scanner.\n", host.AStr(), port)

	snmp, err := gosnmp.NewGoSNMP(host.AStr(), "public", gosnmp.Version2c, 5)
	if err != nil {
		return nil, err
	}

	result := &scanResult{
		host: host,
		svc: &model.Service{
			Timestamp: time.Now(),
			HostID:    host.ID,
			Port:      port,
		},
	}
	// result.Host = *host
	// result.Port = port
	var resStr string
	success := false

	// 3.6.1.2.1.1.1.0
	resp, err := snmp.Get(".1.3.6.1.2.1.1.1.0")
	if err == nil {
	VARLOOP:
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.OctetString:
				resStr = v.Value.(string)
				success = true
				break VARLOOP
			}
		}
	}

	if success {
		result.svc.Response = resStr
		result.svc.Success = true
	}

	return result, nil
} // func (scn *Scanner) scanSNMP(host *model.Host, port uint16) (*scanResult, error)

func (scn *Scanner) scanTelnet(host *model.Host, port uint16) (*scanResult, error) {
	scn.log.Printf("[TRACE] Scanning %s:%d using Telnet scanner.\n", host.AStr(), port)

	var (
		err        error
		n          int
		txtbuf     []byte
		conn       net.Conn
		recvbuffer = make([]byte, 4096)
		probe      = []byte{
			0xff, 0xfc, 0x25, // Won't Authentication
			0xff, 0xfd, 0x03, // Do Suppress Go Ahead
			0xff, 0xfc, 0x18, // Won't Terminal Type
			0xff, 0xfc, 0x1f, // Won't Window Size
			0xff, 0xfc, 0x20, // Won't Terminal Speed
			0xff, 0xfb, 0x22, // Will Linemode
		}
		target = fmt.Sprintf("[%s]:%d", host.AStr(), port)
	)

	if conn, err = net.Dial("tcp", target); err != nil {
		return nil, fmt.Errorf("error connecting to %s:%d - %s",
			host.Name,
			port,
			err.Error())
	}

	defer conn.Close() // nolint: errcheck

	if n, err = conn.Read(recvbuffer); err != nil {
		return nil, fmt.Errorf("error receiving from %s: %s", host.Name, err.Error())
	}

	conn.Write(probe) // nolint: errcheck
	var sndFill int

	for {
		var i int
		sndBuf := make([]byte, 256)
		sndFill = 0

		for i = 0; i < n; i++ {
			if recvbuffer[i] == 0xff {
				sndBuf[sndFill] = 0xff
				sndFill++
				i++
				switch recvbuffer[i] {
				case 0xfb: // WILL
					sndBuf[sndFill] = 0xfe
					sndFill++
				case 0xfd: // DO
					sndBuf[sndFill] = 0xfc
					sndFill++
				}
				i++
				sndBuf[sndFill] = recvbuffer[i]
				sndFill++
			} else if recvbuffer[i] < 0x80 {
				fmt.Printf("Received data from %s: %d/%d\n", host.Name, i, n)
				//return string(recvbuffer[i:n]), nil
				txtbuf = recvbuffer[i:n]
				goto TEXT_FOUND
			}
		}

		if sndFill > 0 {
			_, err = conn.Write(sndBuf[:sndFill])
			if err != nil {
				msg := fmt.Sprintf("Error sending snd_buf to server: %s\n", err.Error())
				fmt.Println(msg)
				return nil, errors.New(msg)
			}
		}

		n, err = conn.Read(recvbuffer)
		if err != nil {
			return nil, fmt.Errorf("error receiving from %s: %s", host.Name, err.Error())
		}

		fmt.Printf("Received %d bytes of data from server.\n", n)
	}

TEXT_FOUND:
	begin := 0
	for ; begin < len(txtbuf); begin++ {
		r, _ := utf8.DecodeRune(txtbuf[begin:begin])
		if txtbuf[begin] >= 0x41 && unicode.IsPrint(r) {
			fmt.Printf("Found Printable character: 0x%02x\n", txtbuf[begin])
			break
		}
	}
	txtbuf = txtbuf[begin:]
	end := 1
	for ; end < len(txtbuf); end++ {
		if txtbuf[end] == 0x00 {
			end--
			txtbuf = txtbuf[:end]
			break
		}
	}
	fmt.Printf("%d bytes of data remaining.\n", len(txtbuf))
	for i := 0; i < len(txtbuf); i++ {
		fmt.Printf("%02d: 0x%02x\n", i, txtbuf[i])
	}

	var result = &scanResult{
		host: host,
		svc: &model.Service{
			HostID:    host.ID,
			Port:      port,
			Timestamp: time.Now(),
			Success:   true,
			Response:  string(txtbuf),
		},
	}
	// result.host = host
	// result.Port = port
	// result.Reply = new(string)
	// *result.Reply = string(txtbuf)
	// result.Stamp = time.Now()
	return result, nil
} // func (scn *Scanner) scanTelnet(host *model.Host, port uint16) (*scanResult, error)
