package xuid

import (
	"github.com/google/uuid"
	"strings"
)

func UUID() string {
	u, _ := uuid.NewRandom()
	return strings.ReplaceAll(u.String(), "-", "")
}
