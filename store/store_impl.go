package store

import (
	"database/sql"
	"demo/model"
	"errors"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

var (
	ErrAnswerExist    = errors.New("an answer with the given key already exists")
	ErrAnswerNotExist = errors.New("no answer with the given key")
)

const dbFilename = "./db"

type storeImpl struct {
	path string
	db   *sql.DB
}

func createDBFileIfNotExists(fileName string) error {
	file, err := os.Create(fileName) // Create SQLite file
	if err != nil && err != os.ErrExist {
		return nil
	}
	return file.Close()
}

func (s *storeImpl) init() error {
	createStmt := `CREATE TABLE IF NOT EXISTS event (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"type" TEXT NOT NULL,
		"key" TEXT,
		"value" TEXT NULL
	  );`

	_, err := s.db.Exec(createStmt)
	return err
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

func (s *storeImpl) insertEvent(t model.EventType, a *model.Answer) error {
	insertStmt := `INSERT INTO event(type, key, value) VALUES (?, ?, ?)`
	_, err := s.db.Exec(insertStmt, t, a.Key, a.Value)
	return err
}

func (s *storeImpl) Create(a *model.Answer) error {
	_, err := s.GetAnswer(a.Key)
	if err == nil {
		return ErrAnswerExist
	}

	if err != ErrAnswerNotExist {
		return err
	}

	return s.insertEvent(model.CreateEvent, a)
}

func (s *storeImpl) Update(a *model.Answer) error {
	_, err := s.GetAnswer(a.Key)
	if err != nil {
		return err
	}
	return s.insertEvent(model.UpdateEvent, a)
}

func (s *storeImpl) Delete(key string) error {
	_, err := s.GetAnswer(key)
	if err != nil {
		return err
	}
	return s.insertEvent(model.DeleteEvent, &model.Answer{Key: key})
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
	query := `SELECT * FROM event WHERE key = (?) ORDER BY id DESC LIMIT 1`
	row := s.db.QueryRow(query, key)

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
	rows, err := s.db.Query(query)

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
