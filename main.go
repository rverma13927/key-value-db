package main

import (

	"kvdb/kv"
	"strconv"
	"sync"
)

func main() {
	db := kv.NewKeyValueDb()
	db.Load()

	//db.Set("Y","1", "1")
	//val, err := db.Get("Y","1")
	//fmt.Println(val, err)

	var wg sync.WaitGroup

	for i := 1; i < 10; i++ {

		wg.Add(2)

		go func() {
			defer wg.Done()
			db.Set("Y","1", strconv.Itoa(i))
		}()
		go func() {
			defer wg.Done()

			db.Get("Y","1")
			//fmt.Println(val, err);
		}()
	}
	wg.Wait()
}