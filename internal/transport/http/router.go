package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	transportui "example.com/taskservice/internal/transport/http/ui"
)

func NewRouter(
	taskHandler *httphandlers.TaskHandler,
	recurringTaskHandler *httphandlers.RecurringTaskHandler,
	docsHandler *swaggerdocs.Handler,
	uiHandler *transportui.Handler,
) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", uiHandler.ServeIndex).Methods(http.MethodGet)
	router.HandleFunc("/ui/{path:.*}", uiHandler.ServeAssets).Methods(http.MethodGet)
	router.HandleFunc("/swagger/openapi.json", docsHandler.ServeSpec).Methods(http.MethodGet)
	router.HandleFunc("/swagger/", docsHandler.ServeUI).Methods(http.MethodGet)
	router.HandleFunc("/swagger", docsHandler.RedirectToUI).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/tasks", taskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", taskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Delete).Methods(http.MethodDelete)
	api.HandleFunc("/recurring-tasks", recurringTaskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/recurring-tasks", recurringTaskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/recurring-tasks/{id:[0-9]+}", recurringTaskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/recurring-tasks/{id:[0-9]+}", recurringTaskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/recurring-tasks/{id:[0-9]+}", recurringTaskHandler.Delete).Methods(http.MethodDelete)

	return router
}
