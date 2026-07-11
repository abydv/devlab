package service

import "errors"

// ErrNotFound is returned by Status or Logs when the Service's
// underlying resource does not exist (Create has not been called yet,
// or Delete has already run).
var ErrNotFound = errors.New("service: not found")
