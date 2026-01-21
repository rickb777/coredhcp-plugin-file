// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

//go:build !cgo

package rangeplugin

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"go.etcd.io/bbolt"
)

func loadDB(path string) (io.Closer, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	//defer db.Close()
	return db, nil
}

// loadRecords loads the DHCPv6/v4 Records global map with records stored on
// the specified file. The records have to be one per line, a mac address and an
// IP address.
func loadRecords(db any) (records map[string]*Record, err error) {
	records = make(map[string]*Record)
	err = db.(*bbolt.DB).View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(bucketName []byte, b *bbolt.Bucket) error {
			pk := keyOf(bucketName)
			if err != nil {
				return err
			}
			return b.ForEach(func(k, v []byte) error {
				hostname := string(k)
				expires := binary.LittleEndian.Uint64(v)
				records[pk.hwaddr.String()] = &Record{
					IP:       pk.ipaddr,
					expires:  int(expires),
					hostname: hostname,
				}
				return nil
			})
		})
	})
	return records, err
}

// saveIPAddress writes out a lease to storage
func saveIPAddress(db any, mac net.HardwareAddr, record *Record) error {
	return db.(*bbolt.DB).Update(func(tx *bbolt.Tx) error {
		pk := key{hwaddr: mac, ipaddr: record.IP}
		b, err := tx.CreateBucketIfNotExists(pk.Bytes())
		if err != nil {
			return err
		}
		expires := make([]byte, 8)
		binary.LittleEndian.PutUint64(expires, uint64(record.expires))
		return b.Put([]byte(record.hostname), expires)
	})
}

// freeIPAddress removes a lease from storage
func freeIPAddress(db any, mac net.HardwareAddr, record *Record) error {
	return db.(*bbolt.DB).Update(func(tx *bbolt.Tx) error {
		pk := key{hwaddr: mac, ipaddr: record.IP}
		return tx.DeleteBucket(pk.Bytes())
	})
}

// key relies on hardware addresses always being 6 bytes
type key struct {
	hwaddr net.HardwareAddr
	ipaddr net.IP
}

func keyOf(bs []byte) key {
	if len(bs) > 6 {
		return key{
			hwaddr: bs[:6],
			ipaddr: bs[6:],
		}
	}
	panic(bs)
}

func (k key) Bytes() []byte {
	bs := make([]byte, 0, len(k.hwaddr)+len(k.ipaddr))
	bs = append(bs, k.hwaddr...)
	bs = append(bs, k.ipaddr...)
	return bs
}
