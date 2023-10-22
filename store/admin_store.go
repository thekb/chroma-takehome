package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"golang.org/x/exp/slices"
)

type adminStore struct {
	userStoreDataSource string
	store               CompoundStore
	usf                 UserStoreFactory
}

var _ AdminStore = (*adminStore)(nil)

func NewAdminStore(ctx context.Context, adminDataSource, userStoreDataSource string, usf UserStoreFactory) (*adminStore, error) {
	store, err := NewSQLite3Store(adminDataSource, time.Now().UnixNano())
	if err != nil {
		return nil, err
	}
	as := &adminStore{
		userStoreDataSource: userStoreDataSource,
		usf:                 usf,
		store:               store,
	}

	err = as.init(ctx)
	if err != nil {
		return nil, err
	}

	return as, nil
}

const (
	global_tables_table          = "global_tables"
	global_permissions           = "global_permissions"
	global_users                 = "global_users"
	global_user_table_permission = "global_user_table_permission"
)

const (
	READ_ALL_PERMISSION         = "READ_ALL"
	WRITE_ALL_PERMISSION        = "WRITE_ALL"
	READ_RESTRICTED_PERMISSION  = "READ_RESTRICTED"
	WRITE_RESTRICTED_PERMISSION = "WRITE_RESTRICTED"
)

var defaultPermissions = func() []string {
	return []string{
		// can read all the records in a table
		READ_ALL_PERMISSION,
		// can create records in a table
		WRITE_ALL_PERMISSION,
		// can read only entries created by the user
		READ_RESTRICTED_PERMISSION,
		// can create new entries and update entries create by the user
		WRITE_RESTRICTED_PERMISSION,
	}
}()

func (s *adminStore) init(ctx context.Context) error {
	if err := s.store.CreateTable(ctx, CreateTableOptions{
		TableName: global_tables_table,
		Definitions: [][]string{
			{"id", "integer", "not null", "primary key"},
			{"name", "text", "not null"},
			{"store_id", "integer", "not null"},
		},
		IfNotExists: true,
	}); err != nil {
		return err
	}
	fmt.Println("created: ", global_tables_table)

	if err := s.store.CreateTable(ctx, CreateTableOptions{
		TableName: global_permissions,
		Definitions: [][]string{
			{"name", "text", "primary key", "not null"},
		},
		IfNotExists: true,
	}); err != nil {
		return err
	}
	fmt.Println("created: ", global_permissions)

	// create default permissions
	for _, perm := range defaultPermissions {

		var exists bool
		res, err := s.store.Query(ctx, QueryOptions{
			TableName:      global_permissions,
			IncludeColumns: []string{"name"},
			Where:          []string{fmt.Sprintf("name = '%s'", perm)},
			Limit:          1,
		})
		if err != nil {
			return err
		}
		if len(res) == 1 {
			exists = true
		}
		if !exists {
			if _, err := s.store.Exec(ctx, ExecOptions{
				Type:      ExecTypeInsert,
				TableName: global_permissions,
				Values: []FieldValue{
					{
						Name:  "name",
						Value: perm,
					},
				},
			}); err != nil {
				return err
			}
		}
		fmt.Println("created: ", perm)

	}

	if err := s.store.CreateTable(ctx, CreateTableOptions{
		TableName: global_users,
		Definitions: [][]string{
			{"id", "integer", "not null", "primary key"},
			{"name", "text"},
			{"token", "text", "not null"},
		},
		IfNotExists: true,
	}); err != nil {
		return err
	}

	fmt.Println("created: ", global_users)

	if err := s.store.CreateTable(ctx, CreateTableOptions{
		TableName: global_user_table_permission,
		Definitions: [][]string{
			{"user_id", "integer", "not null"},
			{"table_name", "text", "not null"},
			{"permission", "text", "not null"},
		},
		IfNotExists: true,
	}); err != nil {
		return err
	}

	fmt.Println("created: ", global_user_table_permission)

	return nil
}

func (s *adminStore) CreateTable(ctx context.Context, opts CreateTableOptions) error {
	//TODO: handle placement of tables on upstream stores
	us, err := s.usf.New(ctx, UserStoreOptions{
		// set id for picking specific upstream stores
		//ID:
		DataSource: s.userStoreDataSource,
	})
	if err != nil {
		return err
	}

	ustc, ok := us.(UserTableCreatorStore)
	if !ok {
		return fmt.Errorf("unable to create table in user store")
	}

	// add created_by column for any new table that is created
	opts.Definitions = append(opts.Definitions, []string{
		"created_by", "integer",
	})

	err = ustc.CreateTable(ctx, opts)
	if err != nil {
		return err
	}

	_, err = s.store.Exec(ctx, ExecOptions{
		Type:      ExecTypeInsert,
		TableName: global_tables_table,
		Values: []FieldValue{
			{
				Name:  "name",
				Value: opts.TableName,
			},
			{
				Name:  "store_id",
				Value: us.ID(),
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *adminStore) AddUser(ctx context.Context, userName string) (string, error) {

	res, err := s.store.Query(ctx, QueryOptions{
		TableName:      global_users,
		IncludeColumns: []string{"name", "token"},
		Where:          []string{fmt.Sprintf("name = '%s'", userName)},
		Limit:          1,
	})
	if err != nil {
		return "", err
	}
	if len(res) == 1 {
		return res[0]["token"].(string), nil
	}
	token := xid.New().String()

	if _, err := s.store.Exec(ctx, ExecOptions{
		Type:      ExecTypeInsert,
		TableName: global_users,
		Values: []FieldValue{
			{
				Name:  "name",
				Value: userName,
			},
			{
				Name:  "token",
				Value: token,
			},
		},
	}); err != nil {
		return "", err
	}

	return token, nil
}

func (s *adminStore) AddPermission(ctx context.Context, userID int64, tableName, permission string) error {

	if !slices.Contains(defaultPermissions, permission) {
		return fmt.Errorf("invalid permission %s. should be one of '%s'", permission, strings.Join(defaultPermissions, ","))
	}

	//TODO: validate token and table exists

	res, err := s.store.Query(ctx, QueryOptions{
		TableName:      global_user_table_permission,
		IncludeColumns: []string{"user_id", "table_name", "permission"},
		Where: []string{
			fmt.Sprintf("user_id = %d", userID),
			fmt.Sprintf("table_name = '%s'", tableName),
			fmt.Sprintf("permission = '%s'", permission),
		},
		Limit: 1,
	})
	if err != nil {
		return err
	}
	if len(res) == 1 {
		return nil
	}

	if _, err := s.store.Exec(ctx, ExecOptions{
		Type:      ExecTypeInsert,
		TableName: global_user_table_permission,
		Values: []FieldValue{
			{
				Name:  "user_id",
				Value: userID,
			},
			{
				Name:  "table_name",
				Value: tableName,
			},
			{
				Name:  "permission",
				Value: permission,
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *adminStore) GetPermissionsForToken(ctx context.Context, token string) ([]TablePermission, error) {

	user, err := s.GetUser(ctx, token)
	if err != nil {
		return nil, err
	}

	res, err := s.store.Query(ctx, QueryOptions{
		TableName:      global_user_table_permission,
		IncludeColumns: []string{"user_id", "table_name", "permission"},
		Where:          []string{fmt.Sprintf("user_id = %d", user.ID)},
	})
	if err != nil {
		return nil, err
	}

	var ret []TablePermission
	for _, rec := range res {
		ret = append(ret, TablePermission{
			TableName:  rec["table_name"].(string),
			Permission: rec["permission"].(string),
		})
	}

	return ret, nil
}

func (s *adminStore) GetUser(ctx context.Context, token string) (*User, error) {
	res, err := s.store.Query(ctx, QueryOptions{
		TableName:      global_users,
		IncludeColumns: []string{"token", "id", "name"},
		Where:          []string{fmt.Sprintf("token = '%s'", token)},
		Limit:          1,
	})
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("user with token '%s' not found", token)
	}

	return &User{
		ID:       res[0]["id"].(int64),
		UserName: res[0]["name"].(string),
	}, nil
}

func (s *adminStore) GetTable(ctx context.Context, tableName string) (*Table, error) {
	res, err := s.store.Query(ctx, QueryOptions{
		TableName:      global_tables_table,
		IncludeColumns: []string{"id", "name", "store_id"},
		Where:          []string{fmt.Sprintf("name = '%s'", tableName)},
		Limit:          1,
	})
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("table with name '%s' not found", tableName)
	}

	return &Table{
		ID:      res[0]["id"].(int64),
		Name:    res[0]["name"].(string),
		StoreID: res[0]["store_id"].(int64),
	}, nil
}
