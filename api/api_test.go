package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/thekb/chroma-takehome/api"
	"github.com/thekb/chroma-takehome/store"
)

const (
	adminToken = "12344567899"
)

func TestStoreAPISimple(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	handler := api.NewStoreHandler(as, ds)
	server := httptest.NewServer(handler)
	defer server.Close()

	e := httpexpect.Default(t, server.URL)

	// add table
	e.POST("/admin/addtable").WithJSON(api.AdminAddTableRequest{
		Token: adminToken,
		CreateTableOptions: store.CreateTableOptions{
			TableName: "foo",
			Definitions: [][]string{
				{"id", "integer", "not null", "primary key"},
				{"name", "text"},
			},
		},
	}).Expect().Status(http.StatusOK).NoContent()

	// add user
	obj := e.POST("/admin/adduser").WithJSON(api.AdminAddUserRequest{
		Token:    adminToken,
		UserName: "test-user",
	}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("userName").IsString().IsEqual("test-user")
	token := obj.Value("userToken").String().Raw()
	// add permissions
	e.POST("/admin/addpermission").WithJSON(api.AdminAddPermissionRequest{
		Token:       adminToken,
		UserName:    "test-user",
		TableName:   "foo",
		Permissions: []string{store.READ_ALL_PERMISSION, store.WRITE_ALL_PERMISSION},
	}).Expect().Status(http.StatusOK).NoContent()
	//store exec
	obj = e.POST("/store/exec").WithJSON(api.StoreExecRequest{
		Token: token,
		ExecOptions: store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: "test",
				},
			},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("lastInsertId").Number().Gt(0)
	//store query
	obj = e.POST("/store/query").WithJSON(api.StoreQueryRequest{
		Token: token,
		QueryOptions: store.QueryOptions{
			TableName:      "foo",
			IncludeColumns: []string{"id", "name"},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("results").Array().NotEmpty()
}

func TestStoreAPIRestricted(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	handler := api.NewStoreHandler(as, ds)
	server := httptest.NewServer(handler)
	defer server.Close()

	e := httpexpect.Default(t, server.URL)

	// add table
	e.POST("/admin/addtable").WithJSON(api.AdminAddTableRequest{
		Token: adminToken,
		CreateTableOptions: store.CreateTableOptions{
			TableName: "foo",
			Definitions: [][]string{
				{"id", "integer", "not null", "primary key"},
				{"name", "text"},
			},
		},
	}).Expect().Status(http.StatusOK).NoContent()

	// add user 1
	obj := e.POST("/admin/adduser").WithJSON(api.AdminAddUserRequest{
		Token:    adminToken,
		UserName: "test-user-1",
	}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("userName").IsString().IsEqual("test-user-1")
	token1 := obj.Value("userToken").String().Raw()
	// add user 2
	obj = e.POST("/admin/adduser").WithJSON(api.AdminAddUserRequest{
		Token:    adminToken,
		UserName: "test-user-2",
	}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("userName").IsString().IsEqual("test-user-2")
	token2 := obj.Value("userToken").String().Raw()

	// add permissions for user 1
	e.POST("/admin/addpermission").WithJSON(api.AdminAddPermissionRequest{
		Token:       adminToken,
		UserName:    "test-user-1",
		TableName:   "foo",
		Permissions: []string{store.READ_RESTRICTED_PERMISSION, store.WRITE_RESTRICTED_PERMISSION},
	}).Expect().Status(http.StatusOK).NoContent()

	// add permissions for user 2
	e.POST("/admin/addpermission").WithJSON(api.AdminAddPermissionRequest{
		Token:       adminToken,
		UserName:    "test-user-2",
		TableName:   "foo",
		Permissions: []string{store.READ_RESTRICTED_PERMISSION, store.WRITE_RESTRICTED_PERMISSION},
	}).Expect().Status(http.StatusOK).NoContent()

	//store exec for user 1
	obj = e.POST("/store/exec").WithJSON(api.StoreExecRequest{
		Token: token1,
		ExecOptions: store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: "test1",
				},
			},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("lastInsertId").Number().Gt(0)
	//store exec for user 2
	obj = e.POST("/store/exec").WithJSON(api.StoreExecRequest{
		Token: token2,
		ExecOptions: store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: "test2",
				},
			},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("lastInsertId").Number().Gt(0)

	//store query for user 1
	obj = e.POST("/store/query").WithJSON(api.StoreQueryRequest{
		Token: token1,
		QueryOptions: store.QueryOptions{
			TableName:      "foo",
			IncludeColumns: []string{"id", "name"},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("results").Array().NotEmpty().Length().IsEqual(1)

	//store query for user 2
	obj = e.POST("/store/query").WithJSON(api.StoreQueryRequest{
		Token: token2,
		QueryOptions: store.QueryOptions{
			TableName:      "foo",
			IncludeColumns: []string{"id", "name"},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("results").Array().NotEmpty().Length().IsEqual(1)

}

func TestStoreAPIReadOnly(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	handler := api.NewStoreHandler(as, ds)
	server := httptest.NewServer(handler)
	defer server.Close()

	e := httpexpect.Default(t, server.URL)

	// add table
	e.POST("/admin/addtable").WithJSON(api.AdminAddTableRequest{
		Token: adminToken,
		CreateTableOptions: store.CreateTableOptions{
			TableName: "foo",
			Definitions: [][]string{
				{"id", "integer", "not null", "primary key"},
				{"name", "text"},
			},
		},
	}).Expect().Status(http.StatusOK).NoContent()

	// add user
	obj := e.POST("/admin/adduser").WithJSON(api.AdminAddUserRequest{
		Token:    adminToken,
		UserName: "test-user",
	}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("userName").IsString().IsEqual("test-user")
	token := obj.Value("userToken").String().Raw()
	// add permissions
	e.POST("/admin/addpermission").WithJSON(api.AdminAddPermissionRequest{
		Token:       adminToken,
		UserName:    "test-user",
		TableName:   "foo",
		Permissions: []string{store.READ_ALL_PERMISSION},
	}).Expect().Status(http.StatusOK).NoContent()
	//store exec
	e.POST("/store/exec").WithJSON(api.StoreExecRequest{
		Token: token,
		ExecOptions: store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: "test",
				},
			},
		},
	}).Expect().Status(http.StatusBadRequest)
	//store query
	obj = e.POST("/store/query").WithJSON(api.StoreQueryRequest{
		Token: token,
		QueryOptions: store.QueryOptions{
			TableName:      "foo",
			IncludeColumns: []string{"id", "name"},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("results").Array().IsEmpty()
}

func TestStoreAPIWriteOnly(t *testing.T) {
	dataSource := ":memory:"
	usf := store.NewUserStoreFactory()

	as, err := store.NewAdminStore(context.TODO(), dataSource, dataSource, usf)
	if err != nil {
		t.Fatal(err)
	}

	ds := store.NewDelegatedStore(as, usf)

	handler := api.NewStoreHandler(as, ds)
	server := httptest.NewServer(handler)
	defer server.Close()

	e := httpexpect.Default(t, server.URL)

	// add table
	e.POST("/admin/addtable").WithJSON(api.AdminAddTableRequest{
		Token: adminToken,
		CreateTableOptions: store.CreateTableOptions{
			TableName: "foo",
			Definitions: [][]string{
				{"id", "integer", "not null", "primary key"},
				{"name", "text"},
			},
		},
	}).Expect().Status(http.StatusOK).NoContent()

	// add user
	obj := e.POST("/admin/adduser").WithJSON(api.AdminAddUserRequest{
		Token:    adminToken,
		UserName: "test-user",
	}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("userName").IsString().IsEqual("test-user")
	token := obj.Value("userToken").String().Raw()
	// add permissions
	e.POST("/admin/addpermission").WithJSON(api.AdminAddPermissionRequest{
		Token:       adminToken,
		UserName:    "test-user",
		TableName:   "foo",
		Permissions: []string{store.WRITE_ALL_PERMISSION},
	}).Expect().Status(http.StatusOK).NoContent()
	//store exec
	obj = e.POST("/store/exec").WithJSON(api.StoreExecRequest{
		Token: token,
		ExecOptions: store.ExecOptions{
			Type:      store.ExecTypeInsert,
			TableName: "foo",
			Values: []store.FieldValue{
				{
					Name:  "name",
					Value: "test",
				},
			},
		},
	}).Expect().Status(http.StatusOK).JSON().Object()
	obj.Value("lastInsertId").Number().Gt(0)
	//store query
	e.POST("/store/query").WithJSON(api.StoreQueryRequest{
		Token: token,
		QueryOptions: store.QueryOptions{
			TableName:      "foo",
			IncludeColumns: []string{"id", "name"},
		},
	}).Expect().Status(http.StatusBadRequest)
}
