package main

import (
	"net"
	"strings"
)

// Get dsn string from config and return slice of dsn struct
func getDsnsFromString(dsnsStr string) (dsns []dsn) {
	if len(dsnsStr) == 0 {
		return
	}

	// clean
	dsnsStr = strings.ToLower(dsnsStr)

	// IP:PORT:TLS
	for _, dsnStr := range strings.Split(dsnsStr, ",") {
		if strings.Count(dsnStr, ":") != 2 {
			ERROR.Fatalln("Bad dsn", dsnStr, " found in config", dsnsStr)
			return
		}
		t := strings.Split(dsnStr, ":")
		// ip & port valid ?
		tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(t[0], t[1]))
		if err != nil {
			ERROR.Fatalln("Bad IP:Port found in dsn", dsnStr, "from config dsn", dsnsStr)
			return
		}
		// tls
		var tls bool
		if t[2] == "0" {
			tls = false
		} else if t[2] == "1" {
			tls = true
		} else {
			ERROR.Fatalln("Bad TLS option found in dsn", dsnStr, "from config dsn", dsnsStr, "Option must be 0 -> no TLS or 1 -> TLS")
		}
		dsns = append(dsns, dsn{*tcpAddr, tls})
		//TRACE.Println(dsns)
	}
	return
}

// Remove trailing and ... brackets (<string> -> string)
func removeBrackets(s string) string {
	if strings.HasPrefix(s, "<") {
		s = s[1:]
	}
	if strings.HasSuffix(s, ">") {
		s = s[0 : len(s)-1]
	}
	return s
}

// Check if a string is in a Slice of string
func isStringInSlice(str string, s []string) (found bool) {
	found = false
	for _, t := range s {
		if t == str {
			found = true
			break
		}
	}
	return
}
