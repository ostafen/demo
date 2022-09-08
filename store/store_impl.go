package store

import (
	"database/sql"
	"errors"
	"os"
	"path"

	"github.com/ostafen/demo/model"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

var (
	ErrAnswerExist    = errors.New("an answer with the given key already exists")
	ErrAnswerNotExist = errors.New("no answer with the given key")
)

const dbFilename = "./data.mysqlite"

type storeImpl struct {
	path string
	db   *sql.DB
}

func createDBFileIfNotExists(fileName string) error {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0666) // Create SQLite file
	if err != nil && err != os.ErrExist {
		return err
	}
	return file.Close()
}

func (s *storeImpl) init() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	createTableStmt := `CREATE TABLE IF NOT EXISTS event (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"type" TEXT NOT NULL,
		"key" TEXT,
		"value" TEXT NULL
	  );`

	_, err = tx.Exec(createTableStmt)
	if err != nil {
		return err
	}

	// since the table can grow a lot, we create an index on the key field to speed-up search queries
	createIndexStmt := `CREATE INDEX IF NOT EXISTS key_index ON event(key);`
	_, err = tx.Exec(createIndexStmt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func Open(dir string) (EventStore, error) {
	dbPath := path.Join(dir, dbFilename)

	if err := createDBFileIfNotExists(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &storeImpl{
		path: dbPath,
		db:   db,
	}

	err = store.init()
	return store, err
}

func (s *storeImpl) insertEvent(t model.EventType, a *model.Answer, txn *sql.Tx) error {
	insertStmt := `INSERT INTO event(type, key, value) VALUES (?, ?, ?)`
	_, err := txn.Exec(insertStmt, t, a.Key, a.Value)
	return err
}

func (s *storeImpl) Create(a *model.Answer) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = s.getAnswer(a.Key, tx)
	if err == nil {
		return ErrAnswerExist
	}

	if err != ErrAnswerNotExist {
		return err
	}

	if err := s.insertEvent(model.CreateEvent, a, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *storeImpl) Update(a *model.Answer) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = s.getAnswer(a.Key, tx)
	if err != nil {
		return err
	}

	if err := s.insertEvent(model.UpdateEvent, a, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *storeImpl) Delete(key string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = s.getAnswer(key, tx)
	if err != nil {
		return err
	}

	if err := s.insertEvent(model.DeleteEvent, &model.Answer{Key: key}, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func scanEvent[T interface{ Scan(dest ...any) error }](row T) (*model.Event, error) {
	var id int
	var evtType, key, value string

	err := row.Scan(&id, &evtType, &key, &value)
	return &model.Event{
		Event: model.EventType(evtType),
		Data:  &model.Answer{Key: key, Value: value},
	}, err
}

func (s *storeImpl) GetAnswer(key string) (*model.Answer, error) {
	txn, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	answ, err := s.getAnswer(key, txn)
	if err != nil {
		return nil, err
	}
	return answ, nil
}

func (s *storeImpl) getAnswer(key string, tx *sql.Tx) (*model.Answer, error) {
	query := `SELECT * FROM event WHERE key = (?) ORDER BY id DESC LIMIT 1`
	row := tx.QueryRow(query, key)

	if row.Err() != nil {
		return nil, row.Err()
	}

	e, err := scanEvent(row)
	if err == sql.ErrNoRows {
		return nil, ErrAnswerNotExist
	}

	if err != nil {
		return nil, err
	}

	if e.Event == model.DeleteEvent {
		return nil, ErrAnswerNotExist
	}

	return e.Data, err
}

func (s *storeImpl) GetHistory(key string) (EventIterator, error) {
	query := `SELECT * FROM event WHERE key = (?) ORDER BY id ASC`
	rows, err := s.db.Query(query, key)

	return &rowIterator{
		rows: rows,
	}, err
}

type rowIterator struct {
	rows *sql.Rows
}

func (it *rowIterator) Next() bool {
	return it.rows.Next()
}

func (it *rowIterator) Value() (*model.Event, error) {
	return scanEvent(it.rows)
}
