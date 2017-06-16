package db

import (
	"errors"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"

	_ "github.com/lib/pq" // Only want to import the interface here
)

const (
	// UnknownClassTable is the name of the table that will hold all the unknown class values provided by Alexa
	UnknownClassTable = "unknown_classes"
	// UnknownItemTable is the name of the table that will hold the unknown item name values passed by Alexa
	UnknownItemTable = "unknown_items"
)

// GetDBConnection is a helper for getting a connection to the DB based on
// environment variables or some other method.
func GetDBConnection() (*gorm.DB, error) {

	//connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=%s",
	//	host, name, user, pass, sslMode)
	db, err := gorm.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println("DB errror: ", err.Error())
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
	defer db.Close()

	rows, err := db.Table("items").
		Select("item_hash").
		Where("item_name = ? AND item_type_name NOT IN ('Material Exchange', '')", itemName).
		Order("max_stack_size DESC").
		Rows()

	if err != nil {
		fmt.Println("Failed to get item hash for name: ", itemName)
		InsertUnknownValueIntoTable(itemName, UnknownItemTable)
		return 0, errors.New("Item lookup failed")
	}

	var hash uint
	if rows.Next() {
		rows.Scan(&hash)
		fmt.Printf("Found hash for item %s: %d\n", itemName, hash)
	}

	if hash == 0 {
		fmt.Println("Didn't find any transferrable items with that name: ", itemName)
		return 0, errors.New("No items founds")
	}

	return hash, nil
}

// InsertUnknownValueIntoTable is a helper method for inserting a value into the specified table.
// This is used when a value for a slot type is not usable. For example when a class name for a character
// is not a valid Destiny class name.
func InsertUnknownValueIntoTable(value, tableName string) {

	conn, err := GetDBConnection()
	if err != nil {
		fmt.Println("Failed to get database connection inserting unknown value!")
		return
	}
	defer conn.Close()

	conn.Exec("INSERT INTO "+tableName+" (value) VALUES(?)", value)
}
