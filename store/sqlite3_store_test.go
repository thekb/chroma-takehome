package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/thekb/chroma-takehome/store"
)

func TestSQLite3Store(t *testing.T) {
	s, err := store.NewSQLite3Store(":memory:", 1)
	if err != nil {
		t.Fatal(err)
	}

	//create table
	err = s.CreateTable(context.TODO(), store.CreateTableOptions{
		TableName: "foo",
		Definitions: [][]string{
			{"id", "integer", "not null", "primary key"},
			{"name", "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		result, err := s.Exec(context.TODO(), store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: fmt.Sprintf("name%d", i),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Log(*result)
	}

	result, err := s.Query(context.TODO(), store.QueryOptions{
		TableName:      "foo",
		IncludeColumns: []string{"id", "name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)

}
