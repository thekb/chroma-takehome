# Requirements
## Functional

1. admin can,
    1. add/delete/update tables
    2. add/delete/update users
    3. add/delete/update permissions -- not implemented
    4. add/delete user permissions on tables

    note: deletes are not implemented

2. permission framework

    READ_ALL -> allow reading all the rows in a table

    WRITE_ALL -> allow inserting to the table, and updating any record in the table

    READ_RESTRICTED -> allow reading only data they have written

    WRITE_RESTRICTED -> allow inserting to table, and updating only records they have inserted

    user permission tuple -> (table_name, user_id, permission)

3. reqular users cannot create/delete tables they can only insert/query/delete from a pre defined table
4. regular users make HTTP request to interact with the store
5. user interaction will follow SQL semanticts
6. should work with SQLite database 

## Non Functional
1. design should be extensible to support other databases in the future
2. permission framework should be extensible, to support more fine grained permissions in the future
3. system should scalable and available  

---
# API Design and Implementation
One of the design trade off to ease the implemetation is avoiding parsing SQL DML/DDL statements. Instead the user input is expected in a pre tokenized format, for example instead of accepting
```sql
SELECT id, name from foo where name = 'test';
```
the API accepts it in a structed format like
```json
{
    "tableName": "foo",
    "includedColumsn": ["id", "name"],
    "where": ["name = 'test'"]
}
```
This simplifies the handling of the user input a great deal. 
Next we abstract the interfacing the a SQL database in the following way,
```go
type QueryOptions struct {
	TableName      string   `json:"tableName"`
	IncludeColumns []string `json:"includeColumns"`
	Where          []string `json:"where"`
	Limit          int      `json:"limit"`
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

type QueryResult []map[string]interface{}
type ExecResult struct {
	LastInsertId int64 `json:"lastInsertId"`
	RowsAffected int64 `json:"rowsAffected"`
}

type UserStore interface {
    Query(context.Context, QueryOptions) (QueryResult, error)
	Exec(context.Context, ExecOptions) (*ExecResult, error)
}

type CreateTableOptions struct {
	TableName   string     `json:"tableName"`
	Definitions [][]string `json:"definitions"`
	IfNotExists bool       `json:"ifNotExists"`
}

type UserTableCreatorStore interface {
	CreateTable(context.Context, CreateTableOptions) error
}
```
This abstract allows to interact with any SQL style database without leaking the implementation details of interacting with the database to the upper layers.

## Implementing UserStore/UserTableCreatorStore interface for SQLite3
We need to convert `QueryOptions`, `ExecOptions` and `CreateTableOptions` into a formatted SQL statement. We use a off the shelf sql builder called `github.com/huandu/go-sqlbuilder` to help with converting the options into formatted SQL query. Then, the formatted query is used to call `QueryContext(ctx context.Context, query string, args ...any) (*Rows, error)` and `ExecContext(ctx context.Context, query string, args ...any) (Result, error)` from golang `database/sql` package from stdlib.

As we don't know the data types of the columns ahead of time we need to handle all the column values as `interface{}`.

`CreateTable` is also implemted in the same way. This gives us all the building blocks required for building our store with access control.

Another tradeoff we made in the interest of time is to not handling insert/update in transactions.

This interface can be extended in the future to support bulk inserts/updates to achieve higher throughput.

## AdminStore and UserStoreFactory
For scaling the service, we need to spread/place the tables (at creation time) on to multiple independepent database nodes. To record the assignment we need to store which table is created in which database node. So we need an helper/indirection instantiating and accessing the store. This is provided by the  `UserStoreFactory`. This indirection prevents coupling the db intialization logic tightly with other components. This acts as a singleton suppling the intialized UserStore when ever it is needed.

```go
type UserStoreOptions struct {
	DataSource string
	ID         int64
}

type UserStoreFactory interface {
	New(ctx context.Context, opts UserStoreOptions) (UserStore, error)
}
```

We need a service which maintains and manages the `tables`, `users` and `permissions`. This is done by `AdminStore`,

```go
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
```
This store acts as the ledger for keeping all the shared state required to enforce access control.

When a table is created using the `AdminStore` it automatically add `created_by` column to the table. This column is used to record the user who has inserted the row. This column is used to enforce `READ_RESTRICTED` and `WRITE_RESTRICTED` access controls. 

Even though the user comes in with a `token`, we record the `id` of the user backing the token and use it for access control as it allows us to revoke/rotate the token in the future, without needing to update update all the columns in all the tables in all the shards. 

When `AdminStore` is initialized we create the necessary tables and default permissions required for book keeping.

## Delegated Store
With the `UserStore` and `AdminStore` in place, we have all the building blocks for enforcing access control. We want to access the `UserStore` as if a user is accessing it, i.e. we need to enforce access control before any operation is performed in the `UserStore`. To help with this we create another abstraction called `DelegatedStore`,
```go
type DelegatedStore interface {
	AsUser(ctx context.Context, opts UserOptions) UserStore
}
```

`DelegatedStore` allows us to access the `UserStore` as if a user is accessing it. It acts a proxy between the user request and the actual `UserStore`, enforcing access control before call the appropriate method in the `UserStore`.

`DelegatedStore` talks to `AdminStore` to fetch the `store_id` of the table we are operating on and uses the `UserStoreFacotry` to create an instance of the `UserStore` for that `store_id`. It then talks to `AdminStore` to fetch the permissions of the user on the given table and enforces access control before calling the appropriate method in `UserStore`.

## User Facing API
We expose two sets of HTTP endpoints, one for admin actions and other for user actions. Admin actions need to supply a harcoded token to establish trust to perform the actions. Users need to supply their token so that access control is enforced.

### Admin Actions
Admin actions are fairly straight forward CRUD operations which are achieved by calling `AdminStore`
### User Actions

User actions are perfomed by using the delegated store by assuing the user. If the user does not have the necessary permissions to perform the action, we return a HTTP 400 BadRequest to the user. Query handler is implemented is shown below,

```go
func storeQuery(ds store.DelegatedStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var opts StoreQueryRequest
		err := json.NewDecoder(r.Body).Decode(&opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

        // assume user by using the user token and perform the
        // action
		results, err := ds.AsUser(r.Context(), store.UserOptions{
			Token: opts.Token,
		}).Query(r.Context(), opts.QueryOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(StoreQueryResponse{
			Results: results,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	}
}

```

Context propagation starts from the incoming HTTP request and go all the way until we hit the data store. This is very imporatant in distributed environment, as it helps us debug and visualize the function calls that span service/network boundaries.

Logging and Metrics are also very important for running any production grade distributed system.

------
RBAC design thoughts

Even though ACL enforcement is done at the user level, configuration is rarely done at user level. Most often ACL configuration is done by grouping the users or assigning labels to users and giving permissions/roles to groups/labels. This allows for a more flexible and easy configurable system. 

Another important thing for a security stand point is the priciple of least privilege (allow list), we should give a user only the permissions they need. How ever in practice, when there are a large number of varied resources, black lists become necessary to reduce the configuration complexity.