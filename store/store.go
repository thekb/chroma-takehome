package store

import (
	"context"
)

type QueryOptions struct {
	TableName      string   `json:"tableName"`
	IncludeColumns []string `json:"includeColumns"`
	Where          []string `json:"where"`
	Limit          int      `json:"limit"`
	//TODO add more query options
}

func (o QueryOptions) Validate() error {
	if o.TableName == "" {
		return NewInvalidQueryOptions("table name is empty")
	}
	return nil
}

type ExecType string

const (
	ExecTypeUpdate = "update"
	ExecTypeInsert = "insert"
)

type FieldValue struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type ExecOptions struct {
	Type      ExecType     `json:"type"`
	TableName string       `json:"tableName"`
	Values    []FieldValue `json:"values"`
	Where     []string     `json:"where"`
}

func (o ExecOptions) Validate() error {
	switch o.Type {
	case ExecTypeInsert, ExecTypeUpdate:
	default:
		return NewInvalidExecOptions("invalid exec type")
	}
	if o.TableName == "" {
		return NewInvalidExecOptions("table name is empty")
	}
	if len(o.Values) == 0 {
		return NewInvalidExecOptions("nothing to update")
	}
	if o.Type == ExecTypeUpdate && len(o.Where) == 0 {
		return NewInvalidExecOptions("update without predicates not allowed")
	}
	return nil
}

type QueryResult []map[string]interface{}
type ExecResult struct {
	LastInsertId int64 `json:"lastInsertId"`
	RowsAffected int64 `json:"rowsAffected"`
}

type CreateTableOptions struct {
	TableName   string     `json:"tableName"`
	Definitions [][]string `json:"definitions"`
	IfNotExists bool       `json:"ifNotExists"`
}

func (o CreateTableOptions) Validate() error {
	if o.TableName == "" {
		return NewInvalidTableCreationOptions("table name is empty")
	}
	return nil
}

type UserStore interface {
	ID() int64
	Query(context.Context, QueryOptions) (QueryResult, error)
	Exec(context.Context, ExecOptions) (*ExecResult, error)
}

type UserTableCreatorStore interface {
	CreateTable(context.Context, CreateTableOptions) error
}

type UserStoreOptions struct {
	DataSource string
	ID         int64
}

type UserStoreFactory interface {
	New(ctx context.Context, opts UserStoreOptions) (UserStore, error)
}

type TablePermission struct {
	TableName  string
	Permission string
}

type User struct {
	ID       int64
	UserName string
}

type Table struct {
	ID      int64
	Name    string
	StoreID int64
}

type AdminStore interface {
	// create a table,
	// equivalent to assigning and creating (DDL) table in a shard
	CreateTable(ctx context.Context, opts CreateTableOptions) error
	// return a created table
	GetTable(ctx context.Context, tableName string) (*Table, error)
	// returns token after adding user successfully
	// returns existing token if user is already present
	AddUser(ctx context.Context, userName string) (string, error)
	// returns user for token
	GetUser(ctx context.Context, token string) (*User, error)
	// no op if (token,table,permission) already exists
	AddPermission(ctx context.Context, userID int64, tableName, permission string) error
	// returns permissing for a token
	GetPermissionsForToken(ctx context.Context, token string) ([]TablePermission, error)
}

type CompoundStore interface {
	UserStore
	UserTableCreatorStore
}

type UserOptions struct {
	Token string
}

type DelegatedStore interface {
	AsUser(ctx context.Context, opts UserOptions) UserStore
}
