package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/rhpds/aws-sandbox/internal/api/v1"
	sandboxdb "github.com/rhpds/aws-sandbox/internal/dynamodb"
	"github.com/rhpds/aws-sandbox/internal/log"
	"go.uber.org/zap"
)

func healthHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

}

// checkEnv checks that the environment variables are set correctly
// and returns an error if not.
func checkEnv() error {
	return nil
}

// GetAccountHandler returns an account
// GET /account
func GetAccountHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")

	// Grab the parameters from Params
	accountName := p.ByName("account")

	// Get the account from DynamoDB
	sandbox, err := sandboxdb.GetAccount(accountName)
	if err != nil {
		if err == sandboxdb.ErrAccountNotFound {
			log.Logger.Warn("GET account", zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			enc.Encode(v1.Error{
				Code:    http.StatusNotFound,
				Message: "Account not found",
			})
			return
		}
		log.Logger.Error("GET account", zap.Error(err))

		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(v1.Error{
			Code:    500,
			Message: "Error reading account",
		})
		return
	}
	// Print account using JSON
	if err := enc.Encode(sandbox); err != nil {
		log.Logger.Error("GET account", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(v1.Error{
			Code:    500,
			Message: "Error reading account",
		})
	}
}

func main() {
	log.InitLoggers(false)

	router := httprouter.New()
	sandboxdb.CheckEnv()
	sandboxdb.SetSession()

	router.GET("/health", healthHandler)
	router.GET("/account/:account", GetAccountHandler)

	log.Err.Fatal(http.ListenAndServe(":8080", router))
}
