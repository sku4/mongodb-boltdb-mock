/*Package db - BoltDB realization */
package db

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/boltdb/bolt"
)

const defaultBucketName = "default"

type Bolt struct {
	db        *bolt.DB
	dir       string //to be deleted on Close()
	bucket    []byte
	key       []byte
	isTempDir bool
}

func (b *Bolt) Connect(resources ...interface{}) (err error) {
	//reading db filename
	boltDBFile, ok := resources[0].(string)
	if !ok {
		return errors.New("Unexpected resources set, want `boltDBFile string`")
	}

	//making directory with the prefix = boltDBFile
	b.dir, ok = resources[1].(string)
	if !ok {
		return errors.New("Unexpected resources set, want `boltDBPath string`")
	}
	if b.dir == "" {
		b.dir, err = ioutil.TempDir("", boltDBFile)
		b.isTempDir = true
	}
	if err != nil {
		return err
	}

	//opening the file
	b.db, err = bolt.Open(fmt.Sprintf("%s/%s", b.dir, boltDBFile), 0644, nil)
	if err != nil {
		return err
	}

	//setting up the buckets (if any received @ resources)
	err = b.db.Update(func(tx *bolt.Tx) error {
		bucketNames := resources[2].([]string)
		for i := 0; i < len(bucketNames); i++ {
			bucketName := bucketNames[i]
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
		}

		//setting up the default bucket
		_, err := tx.CreateBucketIfNotExists([]byte(defaultBucketName))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to set up buckets, %v", err)
	}

	return nil
}

func (b *Bolt) Copy() Handler                                             { return b }
func (b *Bolt) CopyWithSettings(settings ...interface{}) (Handler, error) { return b, nil }
func (b *Bolt) Close() (err error) {
	if err = b.db.Close(); err != nil {
		return
	}
	if b.isTempDir {
		err = os.RemoveAll(b.dir)
	}
	return
}

func (b *Bolt) ExecOn(resources ...interface{}) Querier {
	if resources == nil {
		b.bucket = []byte(defaultBucketName)
		return b
	}
	bucketName := fmt.Sprintf("%s", resources[0])
	if bucketName == "" {
		b.bucket = []byte(defaultBucketName)
		return b
	}
	b.bucket = []byte(bucketName)
	return b
}

func (b *Bolt) Insert(docs ...interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(docs[0])
	if err != nil {
		return fmt.Errorf("Failed to encode to []byte")
	}
	key, err := ioutil.ReadAll(&buf)
	if err != nil {
		return err
	}

	err = enc.Encode(docs[1])
	if err != nil {
		return fmt.Errorf("Failed to encode to []byte, got `%T` as a value", key)
	}
	value, err := ioutil.ReadAll(&buf)
	if err != nil {
		return err
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(b.bucket)
		if bkt == nil {
			return errors.New("No bucket")
		}
		err := bkt.Put(key, value)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
func (b *Bolt) Remove(selector interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(selector)
	if err != nil {
		return fmt.Errorf("Failed to encode selector to []byte")
	}
	key := buf.Bytes()

	err = b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(b.bucket)
		if bkt == nil {
			return errors.New("No bucket")
		}
		err := bkt.Delete(key)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
func (b *Bolt) RemoveAll(selector interface{}) (num int, err error)   { return 0, nil }
func (b *Bolt) Update(selector interface{}, update interface{}) error { return nil }
func (b *Bolt) UpdateAll(selector interface{}, update interface{}) (num int, err error) {
	return 0, nil
}
func (b *Bolt) Upsert(selector interface{}, update interface{}) (num int, err error) {
	return 0, nil
}
func (b *Bolt) Find(query interface{}) Refiner {
	if query != nil {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(query)
		b.key = buf.Bytes()
	}
	return b
}

func (b *Bolt) One(result interface{}) error {
	var buf bytes.Buffer
	dec := gob.NewDecoder(&buf)

	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(b.bucket)
		if bkt == nil {
			return errors.New("No bucket")
		}
		_, err := buf.Write(bkt.Get(b.key))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = dec.Decode(result)
	if err != nil {
		return err
	}

	return nil
}
func (b *Bolt) All(results interface{}) error                 { return nil }
func (b *Bolt) Distinct(key string, result interface{}) error { return nil }
func (b *Bolt) Count() (num int, err error)                   { return 0, nil }
