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

func NewKeyValueDb() *KeyValueDb {
	file, err := os.OpenFile("db.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
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
	for scanner.Scan() {
		line := scanner.Text()

		data := strings.Split(line, ",")

		if data[0] == "SET" {
			t, err := time.Parse(time.RFC3339, data[4])
			_, ok := db.db[data[1]]

			if !ok {
				db.db[data[1]] = make(map[string]Entry)
			}
			if err == nil && t.After(time.Now()) {
				db.db[data[1]][data[2]] = Entry{Value: data[3], ExpireAt: t}
			}
		} else if data[0] == "DELETE" {
			delete(db.db[data[1]], data[2])
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
