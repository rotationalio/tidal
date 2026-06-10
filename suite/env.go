package suite

import (
	"fmt"
	"os"
	"strconv"

	"go.rtnl.ai/x/dsn"
)

// Lookups the database url from the environment. It first tries all the vars in the
// order they are specified. If none are found, it will return the value of the
// DATABASE_URL environment variable. Uses the same semantics as os.LookupEnv.
func DatabaseURL(vars ...string) string {
	for _, v := range vars {
		if value, ok := os.LookupEnv(v); ok && value != "" {
			return value
		}
	}
	return os.Getenv(DATABASE_URL)
}

// Looks up the postgres environment variables and returns a DSN. Returns
// ErrNoDatabaseURL if the variables are not found.
func PostgresEnv() (uri *dsn.DSN, err error) {
	uri = &dsn.DSN{
		Provider: dsn.Postgres,
		User: &dsn.UserInfo{
			Username: os.Getenv("PGUSER"),
			Password: os.Getenv("PGPASSWORD"),
		},
		Host: os.Getenv("PGHOST"),
		Path: os.Getenv("PGDATABASE"),
	}

	// At a minimum, the host must be specified. Defaults are used for the other fields.
	if uri.Host == "" {
		return nil, ErrNoDatabaseURL
	}

	// Try to parse the port number from the environment.
	if port := os.Getenv("PGPORT"); port != "" {
		var portnum uint64
		portnum, err = strconv.ParseUint(port, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("could not parse port number: %q", port)
		}
		uri.Port = uint16(portnum)
	}

	// If the username and password are not specified, then remove the user info.
	if uri.User.Username == "" && uri.User.Password == "" {
		uri.User = nil
	}

	// If the port is not specified, then use the default port.
	if uri.Port == 0 {
		uri.Port = 5432
	}

	// If the path is not specified, then use the default database name.
	if uri.Path == "" {
		uri.Path = "postgres"
	}

	return uri, nil
}
