package data

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

const (
	mysqlDuplicateEntryErrorNumber      uint16 = 1062
	mysqlForeignKeyViolationErrorNumber uint16 = 1452
)

func IsDuplicateError(err error) bool {
	return isMySQLErrorNumber(err, mysqlDuplicateEntryErrorNumber)
}

func IsForeignKeyViolation(err error) bool {
	return isMySQLErrorNumber(err, mysqlForeignKeyViolationErrorNumber)
}

func isMySQLErrorNumber(err error, number uint16) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == number
}
