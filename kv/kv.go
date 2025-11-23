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
	db   map[string]Entry
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
		db:   make(map[string]Entry),
		mu:   sync.RWMutex{},
		file: file,
	}
}

func (db *KeyValueDb) Load() error {
	scanner := bufio.NewScanner(db.file)
	for scanner.Scan() {
		line := scanner.Text()

		data := strings.Split(line, ",")

		if len(data) < 4 {
			continue
		}

		if data[0] == "SET" {
			t, err := time.Parse(time.RFC3339, data[3])
			if err == nil && t.After(time.Now()) {
				db.db[data[1]] = Entry{Value: data[2], ExpireAt: t}
			}
		} else if data[0] == "DELETE" {
			delete(db.db, data[1])
		}
	}
	fmt.Println("Loading Complete")
	return nil
}

func (db *KeyValueDb) Set(key string, value string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	expiry := time.Now().Add(time.Minute * 10)
	line := fmt.Sprintf("SET,%s,%s,%s\n", key, value, expiry.Format(time.RFC3339))
	_, err := db.file.Write([]byte(line))

	if err != nil {
		fmt.Println("Error while writing to file", err)
	}
	fmt.Println("inside Set", value)

	db.db[key] = Entry{Value: value, ExpireAt: expiry}
	return "Value has been set", nil
}

func (db *KeyValueDb) Get(key string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fmt.Println(" Get", db.db[key])
	entry, exists := db.db[key]

	if exists && entry.ExpireAt.Before(time.Now()) {
		return "Key has expired", nil
	}
	if !exists {
		return "", errors.New("key does not exist")
	}
	return entry.Value, nil
}

func (db *KeyValueDb) Delete(key string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	line := "DELETE," + key + "\n"
	db.file.Write([]byte(line))
	delete(db.db, key)
	return "Value has been deleted", nil
}
