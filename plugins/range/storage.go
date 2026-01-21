// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

//go:build cgo

package rangeplugin

import (
	"database/sql"
	"fmt"
	"io"
	"net"

	_ "github.com/mattn/go-sqlite3"
)

func loadDB(path string) (io.Closer, error) {
	// We never close this, but that's ok because plugins are never stopped/unregistered
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%T): %w", err, err)
	}
	if _, err := db.Exec("create table if not exists leases4 (mac string not null, ip string not null, expiry int, hostname string not null, primary key (mac, ip))"); err != nil {
		return nil, fmt.Errorf("table creation failed: %w", err)
	}
	return db, nil
}

// loadRecords loads the DHCPv6/v4 Records global map with records stored on
// the specified file. The records have to be one per line, a mac address and an
// IP address.
func loadRecords(db any) (map[string]*Record, error) {
	return loadRecordsSQL(db.(*sql.DB))
}

func loadRecordsSQL(db *sql.DB) (map[string]*Record, error) {
	rows, err := db.Query("select mac, ip, expiry, hostname from leases4")
	if err != nil {
		return nil, fmt.Errorf("failed to query leases database: %w", err)
	}
	defer rows.Close()
	var (
		mac, ip, hostname string
		expiry            int
		records           = make(map[string]*Record)
	)
	for rows.Next() {
		if err := rows.Scan(&mac, &ip, &expiry, &hostname); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		hwaddr, err := net.ParseMAC(mac)
		if err != nil {
			return nil, fmt.Errorf("malformed hardware address: %s", mac)
		}
		ipaddr := net.ParseIP(ip)
		if ipaddr.To4() == nil {
			return nil, fmt.Errorf("expected an IPv4 address, got: %v", ipaddr)
		}
		records[hwaddr.String()] = &Record{IP: ipaddr, expires: expiry, hostname: hostname}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed lease database row scanning: %w", err)
	}
	return records, nil
}

// saveIPAddress writes out a lease to storage
func saveIPAddress(leasedb any, mac net.HardwareAddr, record *Record) error {
	return saveIPAddressSQL(leasedb.(*sql.DB), mac, record)
}

func saveIPAddressSQL(leasedb *sql.DB, mac net.HardwareAddr, record *Record) error {
	stmt, err := leasedb.Prepare(`insert or replace into leases4(mac, ip, expiry, hostname) values (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("statement preparation failed: %w", err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(
		mac.String(),
		record.IP.String(),
		record.expires,
		record.hostname,
	); err != nil {
		return fmt.Errorf("record insert/update failed: %w", err)
	}
	return nil
}

// freeIPAddress removes a lease from storage
func freeIPAddress(leasedb any, mac net.HardwareAddr, record *Record) error {
	return freeIPAddressSQL(leasedb.(*sql.DB), mac, record)
}

func freeIPAddressSQL(leasedb *sql.DB, mac net.HardwareAddr, record *Record) error {
	stmt, err := leasedb.Prepare(`delete from leases4 where mac = ? and ip = ?`)
	if err != nil {
		return fmt.Errorf("statement preparation failed: %w", err)
	}
	defer stmt.Close()
	if _, err := stmt.Exec(
		mac.String(),
		record.IP.String(),
	); err != nil {
		return fmt.Errorf("record delete failed: %w", err)
	}
	return nil
}
