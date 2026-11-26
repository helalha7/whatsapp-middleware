package httpx

import (
	"fmt"
	"net/http"
	"strconv"
)

func ReadStringQuery(
	r *http.Request,
	key string,
	defaultValue string,
) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func ReadIntQuery(
	r *http.Request,
	key string,
	defaultValue int,
) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("query parameter %q must be an integer", key)
	}
	return parsed, nil
}
