package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/thekb/chroma-takehome/store"
)

// routes
// POST /admin/addtable
// POST /admin/adduser
// POST /admin/addpermission
// POST /store/query
// POST /store/exec

type AdminAddTableRequest struct {
	Token string `json:"token"`
	store.CreateTableOptions
}

type AdminAddUserRequest struct {
	Token    string `json:"token"`
	UserName string `json:"userName"`
}

type AdminAddUserResponse struct {
	UserName  string `json:"userName"`
	UserToken string `json:"userToken"`
}

type AdminAddPermissionRequest struct {
	Token       string   `json:"token"`
	UserName    string   `json:"userName"`
	TableName   string   `json:"tableName"`
	Permissions []string `json:"permissions"`
}

type StoreQueryRequest struct {
	Token string `json:"token"`
	store.QueryOptions
}

type StoreQueryResponse struct {
	Results store.QueryResult `json:"results"`
}

type StoreExecRequest struct {
	Token string `json:"token"`
	store.ExecOptions
}

func NewStoreHandler(as store.AdminStore, ps store.DelegatedStore) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/admin", func(r chi.Router) {
		r.Post("/addtable", adminAddTable(as))
		r.Post("/adduser", adminAddUser(as))
		r.Post("/addpermission", adminAddPermission(as))
	})

	r.Route("/store", func(r chi.Router) {
		r.Post("/query", storeQuery(ps))
		r.Post("/exec", storeExec(ps))
	})

	return r
}

const (
	// harcoding token for now
	adminToken = "12344567899"
)

func adminAddTable(as store.AdminStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req AdminAddTableRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Token != adminToken {
			http.Error(w, "invalid admin token", http.StatusBadRequest)
			return
		}

		err = as.CreateTable(r.Context(), req.CreateTableOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

	}
}

func adminAddUser(as store.AdminStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req AdminAddUserRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Token != adminToken {
			http.Error(w, "invalid admin token", http.StatusBadRequest)
			return
		}

		token, err := as.AddUser(r.Context(), req.UserName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AdminAddUserResponse{
			UserName:  req.UserName,
			UserToken: token,
		})
	}
}

func adminAddPermission(as store.AdminStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req AdminAddPermissionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Token != adminToken {
			http.Error(w, "invalid admin token", http.StatusBadRequest)
			return
		}

		token, err := as.AddUser(r.Context(), req.UserName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := as.GetUser(r.Context(), token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, perm := range req.Permissions {
			err = as.AddPermission(r.Context(), user.ID, req.TableName, perm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func storeQuery(ds store.DelegatedStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var opts StoreQueryRequest
		err := json.NewDecoder(r.Body).Decode(&opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

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

func storeExec(ds store.DelegatedStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var opts StoreExecRequest
		err := json.NewDecoder(r.Body).Decode(&opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := ds.AsUser(r.Context(), store.UserOptions{
			Token: opts.Token,
		}).Exec(r.Context(), opts.ExecOptions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}
