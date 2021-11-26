package graph

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
)

func MarshalUUID(id uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		s := base58.Encode(id[:])
		w.Write([]byte(strconv.Quote(s)))
	})
}

func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.FromBytes(base58.Decode(v))
	default:
		return uuid.UUID{}, fmt.Errorf("%T is not a string", v)
	}
}
