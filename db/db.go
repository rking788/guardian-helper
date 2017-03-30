package db

import (
	"errors"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"

	_ "github.com/lib/pq" // Only want to import the interface here
)

// GetDBConnection is a helper for getting a connection to the DB based on
// environment variables or some other method.
func GetDBConnection() (*gorm.DB, error) {

	// Retrieve required environment variables describing DB connection details
	host := os.Getenv("GUARDIAN_HELPER_DB_HOST")
	name := os.Getenv("GUARDIAN_HELPER_DB_NAME")
	user := os.Getenv("GUARDIAN_HELPER_DB_USER")
	pass := os.Getenv("GUARDIAN_HELPER_DB_PASS")
	sslMode := os.Getenv("GUARDIAN_HELPER_DB_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	if host == "" || name == "" || user == "" || pass == "" {
		return nil, errors.New("Missing one or more DB environment variables")
	}

	connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=%s",
		host, name, user, pass, sslMode)
	db, err := gorm.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// GetItemHashFromName is in charge of querying the database and reading
// the item hash value for the given item name.
func GetItemHashFromName(itemName string) (uint, error) {

	db, err := GetDBConnection()
	if err != nil {
		fmt.Println("Error trying to get connection to DB.")
		return 0, err
	}

	var hash uint
	db.Table("items").
		Select("item_hash").
		Where("item_name = ? AND non_transferrable = ?", itemName, false).
		First(&hash)

	if hash == 0 {
		fmt.Println("Didn't find any transferrable items with that name: ", itemName)
		return 0, errors.New("No items founds")
	}

	return hash, nil
}
