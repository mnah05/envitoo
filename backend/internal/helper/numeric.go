package helper

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// Float64ToNumeric converts a float64 into pgtype.Numeric for NUMERIC columns.
// pgtype.Numeric.Scan only accepts strings, not float64.
func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', 6, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
