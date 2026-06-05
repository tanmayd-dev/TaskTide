package utils

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func StringToUUID(s string) (pgtype.UUID, error) {
	UUID, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{
			Bytes: uuid.Nil,
			Valid: false,
		}, fmt.Errorf("could not parse as UUID: %v", s)
	}
	return pgtype.UUID{
		Bytes: UUID,
		Valid: true,
	}, nil
}
