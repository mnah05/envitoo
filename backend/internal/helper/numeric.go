package helper

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// Float64ToNumeric converts a float64 into pgtype.Numeric for NUMERIC columns.
// pgtype.Numeric.Scan only accepts strings, not float64.
//
// Note 1: pgtype.Numeric.Scan uses the database/sql Scanner interface. That
// interface was designed around text-based database protocols, so it accepts
// string representations of numbers rather than raw float64 values.
//
// Note 2: FormatFloat with 'f' format and 6 decimal places prevents rounding
// artifacts that binary float64 representation can introduce. For example,
// 1.1 cannot be represented exactly as a float64, but "1.100000" is a safe
// round-trip through Postgres NUMERIC(?, 6).
func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', 6, 64)); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
