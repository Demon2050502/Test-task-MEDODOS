package handlers

import (
	"errors"
	"net/http"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	recurringtaskusecase "example.com/taskservice/internal/usecase/recurringtask"
)

type RecurringTaskHandler struct {
	usecase recurringtaskusecase.Usecase
}

func NewRecurringTaskHandler(usecase recurringtaskusecase.Usecase) *RecurringTaskHandler {
	return &RecurringTaskHandler{usecase: usecase}
}

func (h *RecurringTaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req recurringTaskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input, err := req.toCreateInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), input)
	if err != nil {
		writeRecurringUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newRecurringTaskDTO(created))
}

func (h *RecurringTaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	recurringTask, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeRecurringUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurringTaskDTO(recurringTask))
}

func (h *RecurringTaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req recurringTaskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input, err := req.toUpdateInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, input)
	if err != nil {
		writeRecurringUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurringTaskDTO(updated))
}

func (h *RecurringTaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeRecurringUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RecurringTaskHandler) List(w http.ResponseWriter, r *http.Request) {
	recurringTasks, err := h.usecase.List(r.Context())
	if err != nil {
		writeRecurringUsecaseError(w, err)
		return
	}

	response := make([]recurringTaskDTO, 0, len(recurringTasks))
	for i := range recurringTasks {
		response = append(response, newRecurringTaskDTO(&recurringTasks[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func writeRecurringUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, recurringtaskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, recurringtaskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
