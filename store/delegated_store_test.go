package store_test

import (
	"context"
	"testing"

	"github.com/thekb/chroma-takehome/store"
)

func TestDelegatedStoreBasic(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	// create table
	err = as.CreateTable(context.TODO(), store.CreateTableOptions{
		TableName: "foo",
		Definitions: [][]string{
			{"id", "integer", "not null", "primary key"},
			{"name", "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	table, err := as.GetTable(context.TODO(), "foo")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("table:", *table)

	// create user
	token, err := as.AddUser(context.TODO(), "test-user")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("token:", token)

	user, err := as.GetUser(context.TODO(), token)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user.ID, "foo", store.WRITE_ALL_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user.ID, "foo", store.READ_ALL_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	// test inserting data

	_, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token,
	}).Exec(context.TODO(), store.ExecOptions{
		Type:      store.ExecTypeInsert,
		TableName: "foo",
		Values: []store.FieldValue{
			{
				Name:  "name",
				Value: "test",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// test querying without filter
	results, err := ds.AsUser(context.TODO(), store.UserOptions{
		Token: token,
	}).Query(context.TODO(), store.QueryOptions{
		TableName:      "foo",
		IncludeColumns: []string{"id", "name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(results)

	// test update data
	_, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token,
	}).Exec(context.TODO(), store.ExecOptions{
		Type:      store.ExecTypeUpdate,
		TableName: "foo",
		Values: []store.FieldValue{
			{
				Name:  "name",
				Value: "test2",
			},
		},
		Where: []string{"name = 'test'"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// test querying with filter
	results, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token,
	}).Query(context.TODO(), store.QueryOptions{
		TableName:      "foo",
		IncludeColumns: []string{"id", "name"},
		Where:          []string{"name = 'test2'"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(results)

}

func TestDelegatedStoreRestricted(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	// create table
	err = as.CreateTable(context.TODO(), store.CreateTableOptions{
		TableName: "foo",
		Definitions: [][]string{
			{"id", "integer", "not null", "primary key"},
			{"name", "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	table, err := as.GetTable(context.TODO(), "foo")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("table:", *table)

	// create user
	token1, err := as.AddUser(context.TODO(), "test-user-1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("token:", token1)

	user1, err := as.GetUser(context.TODO(), token1)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user1.ID, "foo", store.WRITE_RESTRICTED_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user1.ID, "foo", store.READ_RESTRICTED_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	token2, err := as.AddUser(context.TODO(), "test-user-2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("token:", token1)

	user2, err := as.GetUser(context.TODO(), token2)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user2.ID, "foo", store.WRITE_RESTRICTED_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user2.ID, "foo", store.READ_RESTRICTED_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	// test inserting data

	_, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token1,
	}).Exec(context.TODO(), store.ExecOptions{
		Type:      store.ExecTypeInsert,
		TableName: "foo",
		Values: []store.FieldValue{
			{
				Name:  "name",
				Value: "test1",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token2,
	}).Exec(context.TODO(), store.ExecOptions{
		Type:      store.ExecTypeInsert,
		TableName: "foo",
		Values: []store.FieldValue{
			{
				Name:  "name",
				Value: "test2",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// test querying without filter as user 1
	results, err := ds.AsUser(context.TODO(), store.UserOptions{
		Token: token1,
	}).Query(context.TODO(), store.QueryOptions{
		TableName:      "foo",
		IncludeColumns: []string{"id", "name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(results)

	// test update data
	_, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token1,
	}).Exec(context.TODO(), store.ExecOptions{
		Type:      store.ExecTypeUpdate,
		TableName: "foo",
		Values: []store.FieldValue{
			{
				Name:  "name",
				Value: "test3",
			},
		},
		Where: []string{"name = 'test2'"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// test querying with filter
	results, err = ds.AsUser(context.TODO(), store.UserOptions{
		Token: token1,
	}).Query(context.TODO(), store.QueryOptions{
		TableName:      "foo",
		IncludeColumns: []string{"id", "name"},
		Where:          []string{"name = 'test2'"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(results)

}
