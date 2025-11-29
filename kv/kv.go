package kv

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	Value    string
	ExpireAt time.Time
}

type KeyValueDb struct {
	db   map[string]map[string]Entry // bucket and key-value
	mu   sync.RWMutex
	file *os.File
}

func NewKeyValueDb(filename string) *KeyValueDb {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error opening file", err)
		return nil
	}
	return &KeyValueDb{
		db:   make(map[string]map[string]Entry),
		mu:   sync.RWMutex{},
		file: file,
	}
}

func (db *KeyValueDb) Load() error {
	scanner := bufio.NewScanner(db.file)

	// Buffer for current transaction
	pendingBatch := make(map[string]map[string]Entry)
	inTransaction := false

	for scanner.Scan() {
		line := scanner.Text()

		if line == "TX_BEGIN" {
			inTransaction = true
			pendingBatch = make(map[string]map[string]Entry)
			continue
		}

		if line == "TX_COMMIT" {
			if inTransaction {
				// Apply pending batch to real DB
				for bucket, entries := range pendingBatch {
					if _, ok := db.db[bucket]; !ok {
						db.db[bucket] = make(map[string]Entry)
					}
					for key, entry := range entries {
						db.db[bucket][key] = entry
					}
				}
			}
			inTransaction = false
			pendingBatch = make(map[string]map[string]Entry)
			continue
		}

		data := strings.Split(line, ",")
		if len(data) < 4 {
			continue
		}

		if data[0] == "SET" {
			t, err := time.Parse(time.RFC3339, data[4])
			if err == nil && t.After(time.Now()) {
				entry := Entry{Value: data[3], ExpireAt: t}
				bucket := data[1]
				key := data[2]

				if inTransaction {
					if _, ok := pendingBatch[bucket]; !ok {
						pendingBatch[bucket] = make(map[string]Entry)
					}
					pendingBatch[bucket][key] = entry
				} else {
					// Legacy support (lines outside TX)
					if _, ok := db.db[bucket]; !ok {
						db.db[bucket] = make(map[string]Entry)
					}
					db.db[bucket][key] = entry
				}
			}
		} else if data[0] == "DELETE" {
			bucket := data[1]
			key := data[2]
			if inTransaction {
				if _, ok := pendingBatch[bucket]; ok {
					delete(pendingBatch[bucket], key)
				}
			} else {
				if _, ok := db.db[bucket]; ok {
					delete(db.db[bucket], key)
				}
			}
		}
	}
	fmt.Println("Loading Complete")
	return nil
}

func (db *KeyValueDb) Set(bucket string, key string, value string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	expiry := time.Now().Add(time.Minute * 10)
	line := fmt.Sprintf("SET,%s,%s,%s,%s\n", bucket, key, value, expiry.Format(time.RFC3339))
	_, err := db.file.Write([]byte(line))

	if err != nil {
		fmt.Println("Error while writing to file", err)
	}
	fmt.Println("inside Set", value)
	_, ok := db.db[bucket]

	if !ok {
		db.db[bucket] = make(map[string]Entry)
	}
	db.db[bucket][key] = Entry{Value: value, ExpireAt: expiry}
	return "Value has been set", nil
}

func (db *KeyValueDb) Get(bucket string, key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	_, ok := db.db[bucket]

	if !ok {
		return "", errors.New("Bucket not found")
	}

	fmt.Println(" Get", db.db[bucket][key])
	entry, exists := db.db[bucket][key]

	if exists && entry.ExpireAt.Before(time.Now()) {
		return "Key has expired", nil
	}
	if !exists {
		return "", errors.New("key does not exist")
	}
	return entry.Value, nil
}

func (db *KeyValueDb) Delete(bucket string, key string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	line := "DELETE," + bucket + "," + key + "\n"
	db.file.Write([]byte(line))
	delete(db.db[bucket], key)
	return "Value has been deleted", nil
}
func (db *KeyValueDb) Merge() {
	db.mu.Lock()
	defer db.mu.Unlock()

	temp, err := os.OpenFile("temp.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		fmt.Println("Error while opening temp file", err)
	}

	for key, _ := range db.db {
		for k, v := range db.db[key] {

			if !v.ExpireAt.Before(time.Now()) {
				line := fmt.Sprintf("SET,%s,%s,%s,%s\n", key, k, v.Value, v.ExpireAt.Format(time.RFC3339))
				temp.Write([]byte(line))
			}
		}
	}

	db.file.Close()
	temp.Close()

	os.Rename("temp.log", "db.log")

	db.file, err = os.OpenFile("db.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error opening file", err)
		return
	}
}

func (db *KeyValueDb) Update(fn func(tx *Tx) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tx := &Tx{db: db, pending: make(map[string]map[string]Entry)}

	if err := fn(tx); err != nil {
		return err
	}

	db.file.Write([]byte("TX_BEGIN\n"))

	for bucket, entries := range tx.pending {

		for key, v := range entries {
			if v.ExpireAt.After(time.Now()) {
				line := fmt.Sprintf("SET,%s,%s,%s,%s\n", bucket, key, v.Value, v.ExpireAt.Format(time.RFC3339))
				db.file.Write([]byte(line))
			}
		}
	}

	db.file.Write([]byte("TX_COMMIT\n"))

	for bucket, entries := range tx.pending {
		if _, ok := db.db[bucket]; !ok {
			db.db[bucket] = make(map[string]Entry)
		}
		for k, v := range entries {
			if v.ExpireAt.After(time.Now()) {
				db.db[bucket][k] = v
			}
		}
	}

	return nil
}

type Tx struct {
	db      *KeyValueDb
	pending map[string]map[string]Entry //staging area
}

func (tx *Tx) Set(bucket string, key string, value string) error {
	_, ok := tx.pending[bucket]

	if !ok {
		tx.pending[bucket] = make((map[string]Entry))
	}
	tx.pending[bucket][key] = Entry{Value: value, ExpireAt: time.Now().Add(time.Minute * 10)}
	return nil
}

func (tx *Tx) Get(bucket string, key string) (string, error) {

	_, ok := tx.pending[bucket]

	if !ok {
		return tx.db.Get(bucket, key)
	}

	v := tx.pending[bucket][key]

	if v.ExpireAt.Before(time.Now()) {
		return "", errors.New("Key has expired")
	}
	return v.Value, nil
}
