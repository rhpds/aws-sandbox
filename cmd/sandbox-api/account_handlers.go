package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v4"
	v1 "github.com/rhpds/sandbox/internal/api/v1"
	"github.com/rhpds/sandbox/internal/log"
	"github.com/rhpds/sandbox/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AccountHandler struct {
	AwsAccountProvider models.AwsAccountProvider
	OcpAccountProvider models.OcpAccountProvider
}

func NewAccountHandler(awsAccountProvider models.AwsAccountProvider) *AccountHandler {
	return &AccountHandler{
		AwsAccountProvider: awsAccountProvider,
	}
}

type Account interface {
}

// GetAccountsHandler returns all accounts
// GET /accounts/{kind}
func (h *AccountHandler) GetAccountsHandler(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	serviceUuid := r.URL.Query().Get("service_uuid")
	kind := chi.URLParam(r, "kind")
  if kind == "aws" {
    var (
      err  error
      accounts []models.AwsAccount
    )
    if serviceUuid != "" {
      // Get the account from DynamoDB
      accounts, err = h.AwsAccountProvider.FetchAllByServiceUuid(serviceUuid)

    } else {
      accounts, err = h.AwsAccountProvider.FetchAll()
    }
    if err != nil {
      log.Logger.Error("GET accounts", "error", err)

      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading accounts",
      })
      return
    }

    if len(accounts) == 0 {
      w.WriteHeader(http.StatusNotFound)
    } else {
      w.WriteHeader(http.StatusOK)
    }

    // Print accounts using JSON
    if err := enc.Encode(accounts); err != nil {
      log.Logger.Error("GET accounts", "error", err)
      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading account",
      })
    }
  } else if kind == "ocp" {
		log.Logger.Warn("OCP accounts")
    var (
      err  error
      accounts []models.OcpAccount
    )
    if serviceUuid != "" {
      // Get the account from DynamoDB
      accounts, err = h.OcpAccountProvider.FetchAllByServiceUuid(serviceUuid)

    } else {
      accounts, err = h.OcpAccountProvider.FetchAll()
    }
    if err != nil {
      log.Logger.Error("GET accounts", "error", err)

      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading accounts",
      })
      return
    }

    if len(accounts) == 0 {
      w.WriteHeader(http.StatusNotFound)
    } else {
      w.WriteHeader(http.StatusOK)
    }

    // Print accounts using JSON
    if err := enc.Encode(accounts); err != nil {
      log.Logger.Error("GET accounts", "error", err)
      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading account",
      })
    }
  }

}

// GetAccountHandler returns an account
// GET /accounts/{kind}/{account}
func (h *AccountHandler) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")

	// Grab the parameters from Params
	accountName := chi.URLParam(r, "account")
	kind := chi.URLParam(r, "kind")

	// We don't need 'kind' param for now as it is checked and validated
	// by the swagger openAPI spec.
  if kind == "aws" {
    // Get the account from DynamoDB
    sandbox, err := h.AwsAccountProvider.FetchByName(accountName)
    if err != nil {
      if err == models.ErrAccountNotFound {
        log.Logger.Warn("GET account", "error", err)
        w.WriteHeader(http.StatusNotFound)
        enc.Encode(v1.Error{
          HTTPStatusCode: http.StatusNotFound,
          Message:        "Account not found",
        })
        return
      }
      log.Logger.Error("GET account", "error", err)

      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading account",
      })
      return
    }
    // Print account using JSON
    if err := enc.Encode(sandbox); err != nil {
      log.Logger.Error("GET account", "error", err)
      w.WriteHeader(http.StatusInternalServerError)
      enc.Encode(v1.Error{
        HTTPStatusCode: 500,
        Message:        "Error reading account",
      })
    }
  } else {
      log.Logger.Warn("Implementing")
  }
}

func (h *AccountHandler) CleanupAccountHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the parameters from Params
	accountName := chi.URLParam(r, "account")

	// We don't need 'kind' param for now as it is checked and validated
	// by the swagger openAPI spec.

	// Get the account from DynamoDB
	sandbox, err := h.AwsAccountProvider.FetchByName(accountName)
	if err != nil {
		if err == models.ErrAccountNotFound {
			log.Logger.Warn("GET account", "error", err)
			w.WriteHeader(http.StatusNotFound)
			render.Render(w, r, &v1.Error{
				HTTPStatusCode: http.StatusNotFound,
				Message:        "Account not found",
			})
			return
		}
		log.Logger.Error("GET account", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		render.Render(w, r, &v1.Error{
			HTTPStatusCode: 500,
			Message:        "Error reading account",
		})
		return
	}
	// Mark account for cleanup
	if err := h.AwsAccountProvider.MarkForCleanup(sandbox.Name); err != nil {
		log.Logger.Error("PUT account cleanup", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render.Render(w, r, &v1.Error{
			HTTPStatusCode: 500,
			Message:        "Error marking account for cleanup",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	render.Render(w, r, &v1.SimpleMessage{
		Message: "Account marked for cleanup",
	})
}

func (h *BaseHandler) LifeCycleAccountHandler(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Grab the parameters from Params
		accountName := chi.URLParam(r, "account")

		// We don't need 'kind' param for now as it is checked and validated
		// by the swagger openAPI spec.
		// kind := chi.URLParam(r, "kind")

		reqId := GetReqID(r.Context())

		// Get the account from DynamoDB
		sandbox, err := h.awsAccountProvider.FetchByName(accountName)
		if err != nil {
			if err == models.ErrAccountNotFound {
				log.Logger.Warn("GET account", "error", err)
				w.WriteHeader(http.StatusNotFound)
				render.Render(w, r, &v1.Error{
					HTTPStatusCode: http.StatusNotFound,
					Message:        "Account not found",
				})
				return
			}
			log.Logger.Error("GET account", "error", err)

			w.WriteHeader(http.StatusInternalServerError)
			render.Render(w, r, &v1.Error{
				HTTPStatusCode: 500,
				Message:        "Error reading account",
			})
			return
		}

		// Create a new LifecycleResourceJob
		lifecycleResourceJob := models.LifecycleResourceJob{
			ResourceType: sandbox.Kind,
			ResourceName: sandbox.Name,
			RequestID:    reqId,
			Action:       action,
			Status:       "new",
			DbPool:       h.dbpool,
		}

		// Create job in DB
		if err := lifecycleResourceJob.Create(); err != nil {
			log.Logger.Error("Error creating lifecycle resource job", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			render.Render(w, r, &v1.Error{
				HTTPStatusCode: 500,
				Message:        "Error creating lifecycle resource job",
			})
			return
		}

		// Reply with RequestID
		w.WriteHeader(http.StatusAccepted)
		render.Render(w, r, &v1.LifecycleRequestResponse{
			HTTPStatusCode: http.StatusAccepted,
			Message:        fmt.Sprintf("%s request created", action),
			RequestID:      reqId,
		})
	}
}

func (h *BaseHandler) GetStatusAccountHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the parameters from Params
	accountName := chi.URLParam(r, "account")

	// We don't need 'kind' param for now as it is checked and validated
	// by the swagger openAPI spec.

	// Get the account from DynamoDB
	sandbox, err := h.awsAccountProvider.FetchByName(accountName)
	if err != nil {
		if err == models.ErrAccountNotFound {
			log.Logger.Warn("GET account", "error", err)
			w.WriteHeader(http.StatusNotFound)
			render.Render(w, r, &v1.Error{
				HTTPStatusCode: http.StatusNotFound,
				Message:        "Account not found",
			})
			return
		}
		log.Logger.Error("GET account", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		render.Render(w, r, &v1.Error{
			HTTPStatusCode: http.StatusInternalServerError,
			Message:        "Error reading account",
		})
		return
	}

	// Get the last saved status for that account
	job, err := sandbox.GetLastStatus(h.dbpool)
	if err != nil {
		// Check no row
		if err == pgx.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			render.Render(w, r, &v1.Error{
				HTTPStatusCode: http.StatusNotFound,
				Message:        "Account status not found",
			})
			return
		}

		log.Logger.Error("GET account status", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render.Render(w, r, &v1.Error{
			HTTPStatusCode: http.StatusInternalServerError,
			Message:        "Error getting account status",
		})
		return
	}

	status := models.MakeStatus(job)

	// Print account using JSON
	w.WriteHeader(http.StatusOK)
	log.Logger.Debug("GET account status", "status", job.Result, "updated_at", job.UpdatedAt)
	err = render.Render(w, r, &v1.AccountStatusResponse{
		HTTPStatusCode: http.StatusOK,
		Status:         status,
	})

	if err != nil {
		log.Logger.Error("GET account status", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render.Render(w, r, &v1.Error{
			HTTPStatusCode: http.StatusInternalServerError,
			Message:        "Error getting account status",
		})
		return
	}
}
