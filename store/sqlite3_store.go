package store

import (
	"context"
	"database/sql"

	"github.com/huandu/go-sqlbuilder"

	_ "github.com/mattn/go-sqlite3"
)

type sqlite3Store struct {
	id int64
	db *sql.DB
}

var _ UserStore = (*sqlite3Store)(nil)

func NewSQLite3Store(dataStource string, id int64) (*sqlite3Store, error) {
	db, err := sql.Open("sqlite3", dataStource)
	if err != nil {
		return nil, err
	}

	// set max open conn to 1 to limit concurrency to 1
	db.SetMaxOpenConns(1)

	return &sqlite3Store{db: db, id: id}, nil
}

func (s *sqlite3Store) ID() int64 {
	// TODO: return from store
	return s.id
}

func (s *sqlite3Store) Query(ctx context.Context, opts QueryOptions) (QueryResult, error) {

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	builder := sqlbuilder.SQLite.NewSelectBuilder()

	builder = builder.Select(opts.IncludeColumns...).From(opts.TableName).Where(opts.Where...)
	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}

	query := builder.String()
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// adapted from https://gist.github.com/proprietary/b401b0f7e9fb6c00ed06df553c6a3977
	ret := make([]map[string]interface{}, 0)
	for rows.Next() {
		colVals := make([]interface{}, len(opts.IncludeColumns))
		for i := range colVals {
			colVals[i] = new(interface{})
		}
		err = rows.Scan(colVals...)
		if err != nil {
			return nil, err
		}
		colNames, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		these := make(map[string]interface{})
		for idx, name := range colNames {
			these[name] = *colVals[idx].(*interface{})
		}
		ret = append(ret, these)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *sqlite3Store) Exec(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	switch opts.Type {
	case ExecTypeInsert:
		builder := sqlbuilder.SQLite.NewInsertBuilder()
		builder = builder.InsertInto(opts.TableName)
		var cols []string
		var vals []interface{}
		for _, v := range opts.Values {
			cols = append(cols, v.Name)
			vals = append(vals, v.Value)
		}

		builder = builder.Cols(cols...).Values(vals...)
		query, args = builder.Build()
	case ExecTypeUpdate:
		builder := sqlbuilder.SQLite.NewUpdateBuilder()
		builder = builder.Update(opts.TableName)
		var assigns []string
		for _, v := range opts.Values {
			assigns = append(assigns, builder.Assign(v.Name, v.Value))
		}
		builder.Set(assigns...)

		builder.Where(opts.Where...)
		query, args = builder.Build()
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var lastInsertId int64
	var rowsAffected int64

	switch opts.Type {
	case ExecTypeInsert:
		lastInsertId, err = result.LastInsertId()
		if err != nil {
			return nil, err
		}
	case ExecTypeUpdate:
		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return nil, err
		}
	}

	return &ExecResult{
		LastInsertId: lastInsertId,
		RowsAffected: rowsAffected,
	}, nil
}

var _ UserTableCreatorStore = (*sqlite3Store)(nil)

func (s *sqlite3Store) CreateTable(ctx context.Context, opts CreateTableOptions) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	var query string

	builder := sqlbuilder.SQLite.NewCreateTableBuilder()
	builder = builder.CreateTable(opts.TableName)
	for _, def := range opts.Definitions {
		builder = builder.Define(def...)
	}
	if opts.IfNotExists {
		builder = builder.IfNotExists()
	}

	query = builder.String()

	_, err := s.db.Exec(query)

	return err
}
