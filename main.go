package main

import (
	"fmt"
	"kvdb/kv"
)

func main() {
	db := kv.NewKeyValueDb("db.log")
	db.Load()

	fmt.Println("--- Starting Transaction Test ---")

	// 1. Successful Transaction
	err := db.Update(func(tx *kv.Tx) error {
		fmt.Println("Tx: Setting user:1 and user:2")
		tx.Set("users", "1", "Alice")
		tx.Set("users", "2", "Bob")
		return nil // Commit
	})
	if err != nil {
		fmt.Println("Transaction failed:", err)
	} else {
		fmt.Println("Transaction committed successfully!")
	}

	// Verify Data
	val1, _ := db.Get("users", "1")
	val2, _ := db.Get("users", "2")
	fmt.Println("Result: user:1 =", val1, ", user:2 =", val2)

	// 2. Failed Transaction (Rollback)
	fmt.Println("\n--- Starting Rollback Test ---")
	err = db.Update(func(tx *kv.Tx) error {
		fmt.Println("Tx: Setting user:3 to Charlie")
		tx.Set("users", "3", "Charlie")
		return fmt.Errorf("something went wrong!") // Trigger Rollback
	})

	if err != nil {
		fmt.Println("Transaction rolled back as expected:", err)
	}

	// Verify Data (Should NOT exist)
	val3, err := db.Get("users", "3")
	fmt.Println("Result: user:3 =", val3, "(Error:", err, ")")
}
