package db

import (
	"errors"

	"database/sql"

	"github.com/kpango/glg"
	_ "github.com/lib/pq" // Only want to import the interface here
)

const (
	// UnknownClassTable is the name of the table that will hold all the unknown class values provided by Alexa
	UnknownClassTable = "unknown_classes"
	// UnknownItemTable is the name of the table that will hold the unknown item name values passed by Alexa
	UnknownItemTable = "unknown_items"
)

type LookupDB struct {
	Database          *sql.DB
	HashFromNameStmt  *sql.Stmt
	NameFromHashStmt  *sql.Stmt
	EngramHashStmt    *sql.Stmt
	ItemMetadataStmt  *sql.Stmt
	RandomJokeStmt    *sql.Stmt
	InsertLoadoutStmt *sql.Stmt
	UpdateLoadoutStmt *sql.Stmt
	SelectLoadoutStmt *sql.Stmt
}

var db1 *LookupDB
var dbURL string

// InitEnv provides a package level initialization point for any work that is environment specific
func InitEnv(url string) {
	dbURL = url
}

// InitDatabase is in charge of preparing any Statements that will be commonly used as well
// as setting up the database connection pool.
func InitDatabase() error {

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		glg.Errorf("DB errror: %s", err.Error())
		return err
	}

	stmt, err := db.Prepare("SELECT item_hash FROM items WHERE item_name = $1 AND item_type_name NOT IN ('Material Exchange', '') ORDER BY max_stack_size DESC LIMIT 1")
	if err != nil {
		glg.Errorf("DB error: %s", err.Error())
		return err
	}
	nameFromHashStmt, err := db.Prepare("SELECT item_name FROM items WHERE item_hash = $1 LIMIT 1")
	if err != nil {
		glg.Errorf("DB prepare error: %s", err.Error())
		return err
	}

	// 8 is the item_type value for engrams
	engramHashStmt, err := db.Prepare("SELECT item_hash FROM items WHERE item_name LIKE '%engram%'")
	if err != nil {
		glg.Errorf("DB prepare error: %s", err.Error())
		return err
	}

	itemMetadataStmt, err := db.Prepare("SELECT item_hash, item_name, tier_type, class_type, bucket_type_hash FROM items")
	if err != nil {
		glg.Errorf("DB error: %s", err.Error())
		return err
	}

	randomJokeStmt, err := db.Prepare("SELECT * FROM jokes offset random() * (SELECT COUNT(*) FROM jokes) LIMIT 1")
	if err != nil {
		glg.Errorf("DB prepare error: %s", err.Error())
		return err
	}

	insertLoadoutStmt, err := db.Prepare("INSERT INTO loadouts VALUES ($1,$2,$3)")
	if err != nil {
		glg.Errorf("Error preparing insert loadout statement: %s", err.Error())
		return err
	}

	updateLoadoutStmt, err := db.Prepare("UPDATE loadouts SET loadout=$1 WHERE bungie_membership_id=$2 AND name=$3")
	if err != nil {
		glg.Errorf("Failed preparing update loadout statement: %s", err.Error())
		return err
	}

	selectLoadoutStmt, err := db.Prepare("SELECT loadout FROM loadouts WHERE bungie_membership_id=$1 AND name=$2")
	if err != nil {
		glg.Errorf("Error preparing the select loadout statement: %s", err.Error())
		return err
	}

	db1 = &LookupDB{
		Database:          db,
		HashFromNameStmt:  stmt,
		NameFromHashStmt:  nameFromHashStmt,
		EngramHashStmt:    engramHashStmt,
		ItemMetadataStmt:  itemMetadataStmt,
		RandomJokeStmt:    randomJokeStmt,
		InsertLoadoutStmt: insertLoadoutStmt,
		UpdateLoadoutStmt: updateLoadoutStmt,
		SelectLoadoutStmt: selectLoadoutStmt,
	}

	return nil
}

// GetDBConnection is a helper for getting a connection to the DB based on
// environment variables or some other method.
func GetDBConnection() (*LookupDB, error) {

	if db1 == nil {
		glg.Info("Initializing db!")
		err := InitDatabase()
		if err != nil {
			glg.Errorf("Failed to initialize the database: %s", err.Error())
			return nil, err
		}
	}

	return db1, nil
}

// FindEngramHashes is responsible for querying all of the item_hash values that represent engrams
// and returning them in a map for quick lookup later.
func FindEngramHashes() (map[uint]bool, error) {

	result := make(map[uint]bool)

	db, err := GetDBConnection()
	if err != nil {
		return nil, err
	}

	rows, err := db.EngramHashStmt.Query()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var hash uint
		rows.Scan(&hash)
		result[hash] = true
	}

	return result, nil
}

// LoadItemMetadata will load all rows from the database for all items loaded out of the manifest.
// Only the required columns will be loaded into memory that need to be used later for common operations.
func LoadItemMetadata() (*sql.Rows, error) {

	db, err := GetDBConnection()
	if err != nil {
		return nil, err
	}

	rows, err := db.ItemMetadataStmt.Query()
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// GetItemHashFromName is in charge of querying the database and reading
// the item hash value for the given item name.
func GetItemHashFromName_old(itemName string) (uint, error) {

	db, err := GetDBConnection()
	if err != nil {
		return 0, err
	}

	row := db.HashFromNameStmt.QueryRow(itemName)

	var hash uint
	err = row.Scan(&hash)

	if err == sql.ErrNoRows {
		glg.Warnf("Didn't find any transferrable items with that name: %s", itemName)
		InsertUnknownValueIntoTable(itemName, UnknownItemTable)
		return 0, errors.New("No items found")
	} else if err != nil {
		return 0, errors.New(err.Error())
	}

	return hash, nil
}

// GetItemNameFromHash is in charge of querying the database and reading
// the item name value for the given item hash.
func GetItemNameFromHash(itemHash string) (string, error) {

	db, err := GetDBConnection()
	if err != nil {
		return "", err
	}

	row := db.NameFromHashStmt.QueryRow(itemHash)

	var name string
	err = row.Scan(&name)

	if err == sql.ErrNoRows {
		glg.Warnf("Didn't find any transferrable items with that hash: %s", itemHash)
		return "", errors.New("No items found")
	} else if err != nil {
		return "", errors.New(err.Error())
	}

	return name, nil
}

// SaveLoadout is responsible for persisting the provided serialized loadout to the database.
func SaveLoadout(loadoutJSON []byte, membershipID, name string) error {

	db, err := GetDBConnection()
	if err != nil {
		return err
	}

	_, err = db.InsertLoadoutStmt.Exec(membershipID, name, string(loadoutJSON))

	return err
}

// UpdateLoadout can be used to update an existing loadout by membership ID and loadout name
// this should be used after confirming with the user that they want to update a loadout
// with a spepcific name.
func UpdateLoadout(loadoutJSON []byte, membershipID, name string) error {

	db, err := GetDBConnection()
	if err != nil {
		return err
	}

	_, err = db.UpdateLoadoutStmt.Exec(string(loadoutJSON), membershipID, name)

	return err
}

// SelectLoadout is responsible for querying the database for a loadout with the provided membership ID
// and loadout name. The return value is the JSON string for the loadout requested.
func SelectLoadout(membershipID, name string) (string, error) {

	db, err := GetDBConnection()
	if err != nil {
		return "", err
	}

	row := db.SelectLoadoutStmt.QueryRow(membershipID, name)

	var loadout string
	err = row.Scan(&loadout)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return loadout, nil
}

// InsertUnknownValueIntoTable is a helper method for inserting a value into the specified table.
// This is used when a value for a slot type is not usable. For example when a class name for a character
// is not a valid Destiny class name.
func InsertUnknownValueIntoTable(value, tableName string) {

	conn, err := GetDBConnection()
	if err != nil {
		return
	}

	conn.Database.Exec("INSERT INTO "+tableName+" (value) VALUES(?)", value)
}

// GetRandomJoke will return a setup, punchline, and possibly an error for a random Destiny related joke.
func GetRandomJoke() (string, string, error) {

	db, err := GetDBConnection()
	if err != nil {
		return "", "", err
	}

	row := db.RandomJokeStmt.QueryRow()

	var setup string
	var punchline string
	err = row.Scan(&setup, &punchline)

	return setup, punchline, nil
}
