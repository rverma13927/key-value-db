package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
type Entry struct {
	value    string
	expireAt time.Time
}
type KeyValueDb struct {
	db       map[string]Entry
	mu       sync.RWMutex
	file     *os.File
}

func NewKeyValueDb() *KeyValueDb {
	file, error := os.OpenFile("db.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if error != nil {
		fmt.Println("Error opening file", error)
		return nil
	}
	return &KeyValueDb{
		db:       make(map[string]Entry),
		mu:       sync.RWMutex{},
		file:     file,
	}
}
func (db *KeyValueDb) Load() error {
	scanner := bufio.NewScanner(db.file)
	for scanner.Scan() {
		line := scanner.Text()
	
		data := strings.Split(line, ",")

		if data[0] == "SET" {
			t, err := time.Parse(time.RFC3339, data[3])
			if err == nil && t.After(time.Now()) {
				db.db[data[1]] = Entry{value:data[2],expireAt:	t}
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
	_, error := db.file.Write([]byte(line))

	if error != nil {
		fmt.Println("Error while writing to file", error)
	}
	fmt.Println("inside Set", value)
	
	db.db[key] = Entry{value:value,expireAt:time.Now().Add(time.Minute * 10)}
	return "Value has been set", nil
}

func (db *KeyValueDb) Get(key string) (string, error) {
	db.mu.RLock()
	//If someone is writing... RLock will ensure the reader waits
	// until writer completes so it sees updated value.
	//If multiple readers are reading, then RLock will ensure that no one can write.
	//Readers run in parallel → fast
	//Writers must wait → safe
	//No read/write conflict ever occurs

	// | Operation           | Allowed?   | Why                          |
	// | ------------------- | --------   | ---------------------------- |
	// | **Reader + Reader** | ✔ Yes      | `RLock` allows shared access |
	// | **Writer + Writer** | ❌ No      | `Lock` is exclusive          |
	// | **Reader + Writer** | ❌ No      | Must prevent corrupt data    |

	defer db.mu.RUnlock()

	fmt.Println(" Get", db.db[key])
	entry, exists := db.db[key]
	
	if entry.expireAt.Before(time.Now()) {
		return "Key has expired",nil;
	}
	if !exists {
		return "", errors.New("key does not exist")
	}
	return entry.value, nil
}

func (db *KeyValueDb) Delete(key string) (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	line := "DELETE," + key + "\n"
	db.file.Write([]byte(line))
	delete(db.db, key)
	return "Value has been deleted", nil
}

func main() {
	db := NewKeyValueDb()
	db.Load()

	var wg sync.WaitGroup

	for i := 1; i < 10; i++ {

		wg.Add(2)

		go func() {
			defer wg.Done()
			db.Set("1", strconv.Itoa(i))
		}()
		go func() {
			defer wg.Done()

			db.Get("1")
			//fmt.Println(val, err);
		}()
	}
	wg.Wait()
}
