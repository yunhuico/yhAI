package env

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrNotSet returned when the key of env is not set
	ErrNotSet = errors.New("env not set")
)

// Env is struct of environment variable
type Env string

// ToInt parse value to int
func (e Env) ToInt() (int, error) {
	val, found := lookup(e)
	if !found {
		return 0, ErrNotSet
	}

	i, err := strconv.ParseInt(val, 10, 0)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

// ToBool parse value to bool
func (e Env) ToBool() (bool, error) {
	val, found := lookup(e)
	if !found {
		return false, ErrNotSet
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, err
	}
	return b, nil
}

// ToFloat64 parse value to float64
func (e Env) ToFloat64() (float64, error) {
	val, found := lookup(e)
	if !found {
		return 0, ErrNotSet
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

func (e Env) ToString() (string, error) {
	val, found := lookup(e)
	if !found {
		return "", ErrNotSet
	}
	return val, nil
}

func (e Env) ToStringArr(separator string) (arr []string, err error) {
	val, found := lookup(e)
	if !found {
		err = ErrNotSet
		return
	}

	arr = strings.Split(val, separator)
	return
}

// String convert Env to string
func (e Env) String() string {
	return string(e)
}

func Get(key string) Env {
	return Env(key)
}

func lookup(e Env) (val string, found bool) {
	return os.LookupEnv(e.String())
}

func MustSet(e Env) {
	_, found := lookup(e)
	if !found {
		log.Fatalf("env %s not set\n", e)
	}
}
