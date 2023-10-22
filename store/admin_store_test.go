package store_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/thekb/chroma-takehome/store"
)

func TestAdminStore(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()
	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

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

	token, err := as.AddUser(context.TODO(), "test-user")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("token::", token)
	token, err = as.AddUser(context.TODO(), "test-user")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("token::", token)

	user, err := as.GetUser(context.TODO(), token)
	if err != nil {
		t.Fatal(err)
	}

	err = as.AddPermission(context.TODO(), user.ID, "foo", store.READ_ALL_PERMISSION)
	if err != nil {
		t.Fatal(err)
	}

	tps, err := as.GetPermissionsForToken(context.TODO(), token)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(tps, []store.TablePermission{
		{
			TableName:  "foo",
			Permission: store.READ_ALL_PERMISSION,
		},
	}); diff != "" {
		t.Fatal(diff)
	}

	t.Log(tps)
}
