package httpx

import (
	"log"
	"net/http"
)

func logError(
	r *http.Request,
	err error,
) {
	log.Printf(
		"%s %s %s",
		r.Method, r.URL, err,
	)
}

func ErrorResponse(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	message any,
) {
	env := Envelope{"error": message}

	err := WriteJSON(w, status, env, nil)
	if err != nil {
		logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ServerErrorResponse(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	logError(r, err)
	message := "the server encountered a problem and couldn't process your request"
	ErrorResponse(w, r, http.StatusInternalServerError, message)
}

func ResourceNotFoundResponse(
	w http.ResponseWriter,
	r *http.Request,
) {
	message := "couldn't find the requested resource"
	ErrorResponse(w, r, http.StatusNotFound, message)
}

func DuplicateRecordResponse(
	w http.ResponseWriter,
	r *http.Request,
) {
	message := "a record with these details already exists"
	ErrorResponse(w, r, http.StatusConflict, message)
}

func ValidationFailedResponse(
	w http.ResponseWriter,
	r *http.Request,
	errors map[string]string,
) {
	ErrorResponse(w, r, http.StatusUnprocessableEntity, errors)
}
