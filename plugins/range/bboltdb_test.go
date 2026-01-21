// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

//go:build !cgo

package rangeplugin

import (
	"io"
	"net"
	"os"
	"testing"
)

const testDbName = "test.db"

func testDBSetup(t *testing.T) io.Closer {
	db, err := loadDB(testDbName)
	if err != nil {
		t.Fatalf("Failed to set up test DB: %v", err)
	}

	for _, record := range records {
		mac, _ := net.ParseMAC(record.mac)
		err := saveIPAddress(db, mac, record.ip)
		if err != nil {
			t.Fatalf("failed to insert record into test db: %v", err)
		}
	}
	return db
}

func testDBCleanup(db io.Closer) {
	db.Close()
	os.Remove(testDbName)
}
