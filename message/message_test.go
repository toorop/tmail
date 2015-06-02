package message

import (
	//"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	header1 = `Received: 
	from 209.85.215.52 
	(mail-la0-f52.google.com.) (mail-la0-f52.google.com)       by 5.196.15.145 (mail.tmail.io.) with ESMTPS; 




	  22 May 2015 09:21:35 +0200; tmail 0.0.8; 1887bff38a4d7f7c0fff14f82cdb3f0054c9caf4





	  `
)

func Test_FoldHeader(t *testing.T) {
	header := []byte(header1)
	FoldHeader(&header)
	println(string(header))
	assert.NotEmpty(t, header)
}
