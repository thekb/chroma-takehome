package store

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
)

type delegatedStore struct {
	as  AdminStore
	uo  UserOptions
	usf UserStoreFactory
}

var _ DelegatedStore = (*delegatedStore)(nil)

func NewDelegatedStore(as AdminStore, usf UserStoreFactory) *delegatedStore {
	return &delegatedStore{
		as:  as,
		usf: usf,
	}
}

func (s *delegatedStore) AsUser(ctx context.Context, opts UserOptions) UserStore {

	return &delegatedStore{
		as:  s.as,
		uo:  opts,
		usf: s.usf,
	}
}

var _ UserStore = (*delegatedStore)(nil)

func (s *delegatedStore) ID() int64 {
	return -1
}

func (s *delegatedStore) Query(ctx context.Context, opts QueryOptions) (QueryResult, error) {

	user, err := s.as.GetUser(ctx, s.uo.Token)
	if err != nil {
		return nil, err
	}

	table, err := s.as.GetTable(ctx, opts.TableName)
	if err != nil {
		return nil, err
	}

	perms, err := s.as.GetPermissionsForToken(ctx, s.uo.Token)
	if err != nil {
		return nil, err
	}

	var tablePerms []string

	for _, perm := range perms {
		if opts.TableName == perm.TableName {
			tablePerms = append(tablePerms, perm.Permission)
		}
	}

	if len(tablePerms) == 0 {
		return nil, fmt.Errorf("user with token '%s' not allowed to access table '%s'", s.uo.Token, opts.TableName)
	}

	hasReadPermission := func() bool {
		return slices.ContainsFunc(tablePerms, func(s string) bool {
			if s == READ_ALL_PERMISSION || s == READ_RESTRICTED_PERMISSION {
				return true
			}
			return false
		})
	}

	// assumption if user has both READ_ALL and READ_RESTRICTED then user will have READ_ALL
	hasOnlyRestrictedReadPermission := func() bool {
		return slices.Contains(tablePerms, READ_RESTRICTED_PERMISSION) && !slices.Contains(tablePerms, READ_ALL_PERMISSION)
	}

	if !hasReadPermission() {
		return nil, fmt.Errorf("user with '%s' token cannot perform this query action", s.uo.Token)
	}

	if hasOnlyRestrictedReadPermission() {
		opts.Where = append(opts.Where, fmt.Sprintf("created_by = %d", user.ID))
	}

	us, err := s.usf.New(ctx, UserStoreOptions{
		ID: table.StoreID,
	})
	if err != nil {
		return nil, err
	}

	return us.Query(ctx, opts)
}

func (s *delegatedStore) Exec(ctx context.Context, opts ExecOptions) (*ExecResult, error) {
	user, err := s.as.GetUser(ctx, s.uo.Token)
	if err != nil {
		return nil, err
	}

	table, err := s.as.GetTable(ctx, opts.TableName)
	if err != nil {
		return nil, err
	}

	perms, err := s.as.GetPermissionsForToken(ctx, s.uo.Token)
	if err != nil {
		return nil, err
	}

	var tablePerms []string

	for _, perm := range perms {
		if opts.TableName == perm.TableName {
			tablePerms = append(tablePerms, perm.Permission)
		}
	}

	if len(tablePerms) == 0 {
		return nil, fmt.Errorf("user with token '%s' not allowed to access table '%s'", s.uo.Token, opts.TableName)
	}

	hasWritePermission := func() bool {
		return slices.ContainsFunc(tablePerms, func(s string) bool {
			if s == WRITE_ALL_PERMISSION || s == WRITE_RESTRICTED_PERMISSION {
				return true
			}
			return false
		})
	}

	// assumption if user have both WRITE_RESTRICTED and WRITE_ALL, then they will have WRITE_ALL
	hasOnlyRestrictedUpdatePermission := func() bool {
		return slices.Contains(tablePerms, WRITE_RESTRICTED_PERMISSION) && !slices.Contains(tablePerms, WRITE_ALL_PERMISSION)
	}

	if !hasWritePermission() {
		return nil, fmt.Errorf("user with token '%s' cannot perform exec action", s.uo.Token)
	}

	switch opts.Type {
	case ExecTypeInsert:
		opts.Values = append(opts.Values, FieldValue{
			Name:  "created_by",
			Value: user.ID,
		})
	case ExecTypeUpdate:
		if hasOnlyRestrictedUpdatePermission() {
			opts.Where = append(opts.Where, fmt.Sprintf("created_by = %d", user.ID))
		}
	}

	us, err := s.usf.New(ctx, UserStoreOptions{
		ID: table.StoreID,
	})
	if err != nil {
		return nil, err
	}

	return us.Exec(ctx, opts)
}
