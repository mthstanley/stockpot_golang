package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func convertText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func convertToText(t *string) pgtype.Text {
	if t == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *t, Valid: true}
}

func convertInt4(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

func convertToInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

func convertEpochSeconds(t pgtype.Numeric) *time.Duration {
	if !t.Valid {
		return nil
	}

	v, err := t.Int64Value()
	if err != nil {
		fmt.Printf("%v", err)
		return nil
	}

	d := time.Duration(v.Int64) * time.Second
	return &d
}

func convertToInterval(t *time.Duration) pgtype.Interval {
	if t == nil {
		return pgtype.Interval{Valid: false}
	}
	return pgtype.Interval{Microseconds: t.Microseconds(), Valid: true}
}
